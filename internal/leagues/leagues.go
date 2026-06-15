// Package leagues provides the private-competition ("League") endpoints:
// create (with a unique invite code, creator auto-joined as owner), join by
// code, list mine, and a leaderboard. Scoring totals are filled by the Phase 5
// engine; until then the leaderboard returns members with zeroed points.
package leagues

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/scoring"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars
const botEmailSuffix = "@dev.local"
const leagueInvitesCollection = "league_invites"

const (
	inviteStatusPending  = "pending"
	inviteStatusAccepted = "accepted"
	inviteStatusDeclined = "declined"
)

// GlobalInviteCode is the fixed invite code of the auto-managed "Global" league
// that every registered user belongs to.
const GlobalInviteCode = "GLOBAL"

func uniqueCode(app core.App) string {
	for attempt := 0; attempt < 10; attempt++ {
		candidate := newInviteCode(6)
		if _, err := app.FindFirstRecordByFilter("leagues", "inviteCode = {:c}", map[string]any{"c": candidate}); err != nil {
			return candidate
		}
	}
	return ""
}

func newInviteCode(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	var sb strings.Builder
	for _, v := range b {
		sb.WriteByte(codeAlphabet[int(v)%len(codeAlphabet)])
	}
	return sb.String()
}

func bad(e *core.RequestEvent, code int, msg string) error {
	return e.JSON(code, map[string]string{"error": msg})
}

type inviteUserDTO struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email,omitempty"`
	AvatarURL *string `json:"avatarUrl"`
}

type leagueInviteDTO struct {
	ID          string        `json:"id"`
	LeagueID    string        `json:"leagueId"`
	LeagueName  string        `json:"leagueName"`
	InvitedUser inviteUserDTO `json:"invitedUser"`
	InvitedBy   inviteUserDTO `json:"invitedBy"`
	Status      string        `json:"status"`
	Created     string        `json:"created"`
	Updated     string        `json:"updated"`
	ActedAt     string        `json:"actedAt,omitempty"`
}

func recordDate(rec *core.Record, field string) string {
	dt := rec.GetDateTime(field)
	if dt.IsZero() {
		return ""
	}
	return dt.Time().Format(time.RFC3339Nano)
}

func leagueAvatarURL(user *core.Record) *string {
	file := user.GetString("avatar")
	if file == "" {
		return nil
	}
	url := "/api/files/users/" + user.Id + "/" + file
	return &url
}

func inviteUserInfo(user *core.Record, includeEmail bool) inviteUserDTO {
	name := strings.TrimSpace(user.GetString("name"))
	if name == "" {
		name = "Spelar"
	}
	dto := inviteUserDTO{
		ID:        user.Id,
		Name:      name,
		AvatarURL: leagueAvatarURL(user),
	}
	if includeEmail {
		dto.Email = user.GetString("email")
	}
	return dto
}

func isAppSuperuser(app core.App, user *core.Record) bool {
	if user == nil {
		return false
	}
	email := strings.ToLower(strings.TrimSpace(user.GetString("email")))
	if email == "" {
		return false
	}
	superusers, err := app.FindCollectionByNameOrId("_superusers")
	if err != nil {
		return false
	}
	_, err = app.FindAuthRecordByEmail(superusers, email)
	return err == nil
}

func ownedLeague(app core.App, e *core.RequestEvent, leagueID string) (*core.Record, error) {
	league, err := app.FindRecordById("leagues", leagueID)
	if err != nil {
		return nil, bad(e, http.StatusNotFound, "league not found")
	}
	if league.GetString("inviteCode") == GlobalInviteCode {
		return nil, bad(e, http.StatusForbidden, "global league cannot be managed")
	}
	if league.GetString("owner") != e.Auth.Id {
		return nil, bad(e, http.StatusForbidden, "only the league owner can do this")
	}
	return league, nil
}

func requireInviteManager(app core.App, e *core.RequestEvent, leagueID string) (*core.Record, error) {
	league, err := app.FindRecordById("leagues", leagueID)
	if err != nil {
		return nil, bad(e, http.StatusNotFound, "league not found")
	}
	if league.GetString("inviteCode") == GlobalInviteCode {
		return nil, bad(e, http.StatusForbidden, "global league cannot be invited to")
	}
	if league.GetString("owner") == e.Auth.Id {
		return league, nil
	}
	if !isAppSuperuser(app, e.Auth) {
		return nil, bad(e, http.StatusForbidden, "owner access required")
	}
	if _, err := app.FindFirstRecordByFilter("league_members",
		"league = {:l} && user = {:u}",
		map[string]any{"l": leagueID, "u": e.Auth.Id}); err != nil {
		return nil, bad(e, http.StatusForbidden, "not a member of this league")
	}
	return league, nil
}

func userIDSet(records []*core.Record, field string) map[string]bool {
	out := make(map[string]bool, len(records))
	for _, rec := range records {
		id := rec.GetString(field)
		if id != "" {
			out[id] = true
		}
	}
	return out
}

func pendingInviteUserIDs(app core.App, leagueID string) (map[string]bool, error) {
	pending, err := app.FindRecordsByFilter(leagueInvitesCollection,
		"league = {:l} && status = {:s}", "", 0, 0,
		map[string]any{"l": leagueID, "s": inviteStatusPending})
	if err != nil {
		return nil, err
	}
	return userIDSet(pending, "invitedUser"), nil
}

func leagueMemberIDs(app core.App, leagueID string) (map[string]bool, error) {
	members, err := app.FindRecordsByFilter("league_members",
		"league = {:l}", "", 0, 0, map[string]any{"l": leagueID})
	if err != nil {
		return nil, err
	}
	return userIDSet(members, "user"), nil
}

func leagueInviteInfo(app core.App, invite *core.Record, includeInvitedEmail bool) (leagueInviteDTO, error) {
	league, err := app.FindRecordById("leagues", invite.GetString("league"))
	if err != nil {
		return leagueInviteDTO{}, err
	}
	invited, err := app.FindRecordById("users", invite.GetString("invitedUser"))
	if err != nil {
		return leagueInviteDTO{}, err
	}
	inviter, err := app.FindRecordById("users", invite.GetString("invitedBy"))
	if err != nil {
		return leagueInviteDTO{}, err
	}
	return leagueInviteDTO{
		ID:          invite.Id,
		LeagueID:    league.Id,
		LeagueName:  league.GetString("name"),
		InvitedUser: inviteUserInfo(invited, includeInvitedEmail),
		InvitedBy:   inviteUserInfo(inviter, false),
		Status:      invite.GetString("status"),
		Created:     recordDate(invite, "created"),
		Updated:     recordDate(invite, "updated"),
		ActedAt:     recordDate(invite, "actedAt"),
	}, nil
}

// Register wires the League endpoints. Most require an authenticated user;
// the invite-preview route below is intentionally public.
func Register(app core.App, se *core.ServeEvent) {
	// Auto-managed "Global" league: ensure it exists, backfill existing users,
	// and add every new user as a member when their account is created.
	if err := backfillGlobal(app); err != nil {
		log.Printf("[leagues] global backfill failed: %v", err)
	}
	app.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		if isBotUser(e.Record) {
			return e.Next()
		}
		if err := ensureGlobalMember(e.App, e.Record.Id); err != nil {
			log.Printf("[leagues] auto-join global failed for %s: %v", e.Record.Id, err)
		}
		return e.Next()
	})

	// Public: resolve an invite code to a league name for the invite landing
	// page. Possessing the code is the capability (it's an invite link); only
	// id + name are exposed, nothing member- or score-related.
	//
	// Lives under /api/invite (not /api/leagues) on purpose: Go 1.22's router
	// rejects a path-param route under /api/leagues/ as ambiguous against
	// /api/leagues/{id}/leaderboard.
	se.Router.GET("/api/invite/{code}", func(e *core.RequestEvent) error {
		code := strings.ToUpper(strings.TrimSpace(e.Request.PathValue("code")))
		league, err := app.FindFirstRecordByFilter("leagues",
			"inviteCode = {:c}", map[string]any{"c": code})
		if err != nil {
			return bad(e, http.StatusNotFound, "invalid invite code")
		}
		return e.JSON(http.StatusOK, map[string]any{
			"id": league.Id, "name": league.GetString("name"),
		})
	})

	g := se.Router.Group("/api/leagues")
	g.Bind(apis.RequireAuth())

	// POST /api/leagues/create  { "name": "..." }
	g.POST("/create", func(e *core.RequestEvent) error {
		var body struct {
			Name string `json:"name"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return bad(e, http.StatusBadRequest, "name required")
		}

		col, err := app.FindCollectionByNameOrId("leagues")
		if err != nil {
			return err
		}

		code := uniqueCode(app)
		if code == "" {
			return bad(e, http.StatusInternalServerError, "could not generate invite code")
		}

		def, _ := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")

		league := core.NewRecord(col)
		league.Set("name", name)
		league.Set("inviteCode", code)
		league.Set("owner", e.Auth.Id)
		if def != nil {
			league.Set("scoringConfig", def.Id)
		}
		if err := app.Save(league); err != nil {
			return err
		}
		if err := addMember(app, league.Id, e.Auth.Id, "owner"); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{
			"id": league.Id, "name": name, "inviteCode": code,
		})
	})

	// POST /api/leagues/join  { "code": "ABC123" }
	g.POST("/join", func(e *core.RequestEvent) error {
		var body struct {
			Code string `json:"code"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		code := strings.ToUpper(strings.TrimSpace(body.Code))
		league, err := app.FindFirstRecordByFilter("leagues", "inviteCode = {:c}", map[string]any{"c": code})
		if err != nil {
			return bad(e, http.StatusNotFound, "invalid invite code")
		}
		if existing, _ := app.FindFirstRecordByFilter("league_members",
			"league = {:l} && user = {:u}",
			map[string]any{"l": league.Id, "u": e.Auth.Id}); existing != nil {
			return e.JSON(http.StatusOK, map[string]any{"id": league.Id, "name": league.GetString("name"), "already": true})
		}
		if err := addMember(app, league.Id, e.Auth.Id, "member"); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"id": league.Id, "name": league.GetString("name")})
	})

	// DELETE /api/leagues/{id}
	g.DELETE("/{id}", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		league, err := app.FindRecordById("leagues", id)
		if err != nil {
			return bad(e, http.StatusNotFound, "league not found")
		}
		if league.GetString("inviteCode") == GlobalInviteCode {
			return bad(e, http.StatusForbidden, "global league cannot be deleted")
		}
		if league.GetString("owner") != e.Auth.Id {
			return bad(e, http.StatusForbidden, "only the owner can delete this league")
		}
		if err := app.Delete(league); err != nil {
			return err
		}
		return e.NoContent(http.StatusNoContent)
	})

	// GET /api/leagues/mine
	g.GET("/mine", func(e *core.RequestEvent) error {
		members, err := app.FindRecordsByFilter("league_members",
			"user = {:u}", "-joinedAt", 0, 0, map[string]any{"u": e.Auth.Id})
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0, len(members))
		for _, m := range members {
			lg, err := app.FindRecordById("leagues", m.GetString("league"))
			if err != nil {
				continue
			}
			cnt, _ := app.CountRecords("league_members",
				dbx.HashExp{"league": lg.Id})
			role := m.GetString("role")
			private := lg.GetBool("privateCode")
			code := lg.GetString("inviteCode")
			if private && role != "owner" {
				code = ""
			}
			out = append(out, map[string]any{
				"id":         lg.Id,
				"name":       lg.GetString("name"),
				"inviteCode": code,
				"role":       role,
				"private":    private,
				"members":    cnt,
			})
		}
		return e.JSON(http.StatusOK, map[string]any{"leagues": out})
	})

	// POST /api/leagues/{id}/rename  { "name": "..." }
	g.POST("/{id}/rename", func(e *core.RequestEvent) error {
		league, err := ownedLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return bad(e, http.StatusBadRequest, "name required")
		}
		league.Set("name", name)
		if err := app.Save(league); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"id": league.Id, "name": name})
	})

	// POST /api/leagues/{id}/code/regenerate
	g.POST("/{id}/code/regenerate", func(e *core.RequestEvent) error {
		league, err := ownedLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		code := uniqueCode(app)
		if code == "" {
			return bad(e, http.StatusInternalServerError, "could not generate invite code")
		}
		league.Set("inviteCode", code)
		if err := app.Save(league); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"inviteCode": code})
	})

	// POST /api/leagues/{id}/code/visibility  { "private": true }
	g.POST("/{id}/code/visibility", func(e *core.RequestEvent) error {
		league, err := ownedLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			Private bool `json:"private"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		league.Set("privateCode", body.Private)
		if err := app.Save(league); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"private": body.Private})
	})

	// POST /api/leagues/{id}/members/remove  { "userId": "..." }
	g.POST("/{id}/members/remove", func(e *core.RequestEvent) error {
		league, err := ownedLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			UserID string `json:"userId"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		userID := strings.TrimSpace(body.UserID)
		if userID == "" {
			return bad(e, http.StatusBadRequest, "userId required")
		}
		if userID == league.GetString("owner") {
			return bad(e, http.StatusBadRequest, "the owner cannot be removed")
		}
		member, err := app.FindFirstRecordByFilter("league_members",
			"league = {:l} && user = {:u}",
			map[string]any{"l": league.Id, "u": userID})
		if err != nil {
			return bad(e, http.StatusNotFound, "not a member of this league")
		}
		if err := app.Delete(member); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})

	// GET /api/leagues/{id}/invite-candidates?q=...
	g.GET("/{id}/invite-candidates", func(e *core.RequestEvent) error {
		leagueID := e.Request.PathValue("id")
		if _, err := requireInviteManager(app, e, leagueID); err != nil {
			return err
		}
		q := strings.ToLower(strings.TrimSpace(e.Request.URL.Query().Get("q")))
		if len([]rune(q)) < 2 {
			return e.JSON(http.StatusOK, map[string]any{"users": []inviteUserDTO{}})
		}

		memberIDs, err := leagueMemberIDs(app, leagueID)
		if err != nil {
			return err
		}
		pendingIDs, err := pendingInviteUserIDs(app, leagueID)
		if err != nil {
			return err
		}
		users, err := app.FindRecordsByFilter("users", "id != ''", "name", 0, 0)
		if err != nil {
			return err
		}
		sort.SliceStable(users, func(i, j int) bool {
			ai := strings.ToLower(strings.TrimSpace(users[i].GetString("name") + " " + users[i].GetString("email")))
			aj := strings.ToLower(strings.TrimSpace(users[j].GetString("name") + " " + users[j].GetString("email")))
			return ai < aj
		})
		out := make([]inviteUserDTO, 0, 12)
		for _, user := range users {
			if len(out) >= 12 {
				break
			}
			if user.Id == e.Auth.Id || memberIDs[user.Id] || pendingIDs[user.Id] || isBotUser(user) {
				continue
			}
			haystack := strings.ToLower(user.GetString("name") + " " + user.GetString("email"))
			if strings.Contains(haystack, q) {
				out = append(out, inviteUserInfo(user, true))
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"users": out})
	})

	// GET /api/leagues/{id}/invites
	g.GET("/{id}/invites", func(e *core.RequestEvent) error {
		leagueID := e.Request.PathValue("id")
		if _, err := requireInviteManager(app, e, leagueID); err != nil {
			return err
		}
		recs, err := app.FindRecordsByFilter(leagueInvitesCollection,
			"league = {:l} && status = {:s}", "-created", 0, 0,
			map[string]any{"l": leagueID, "s": inviteStatusPending})
		if err != nil {
			return err
		}
		out := make([]leagueInviteDTO, 0, len(recs))
		for _, rec := range recs {
			item, err := leagueInviteInfo(app, rec, true)
			if err != nil {
				continue
			}
			out = append(out, item)
		}
		return e.JSON(http.StatusOK, map[string]any{"invites": out})
	})

	// POST /api/leagues/{id}/invites  { "userId": "..." }
	g.POST("/{id}/invites", func(e *core.RequestEvent) error {
		leagueID := e.Request.PathValue("id")
		if _, err := requireInviteManager(app, e, leagueID); err != nil {
			return err
		}
		var body struct {
			UserID string `json:"userId"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		userID := strings.TrimSpace(body.UserID)
		if userID == "" {
			return bad(e, http.StatusBadRequest, "user required")
		}
		if userID == e.Auth.Id {
			return bad(e, http.StatusBadRequest, "cannot invite yourself")
		}
		user, err := app.FindRecordById("users", userID)
		if err != nil || isBotUser(user) {
			return bad(e, http.StatusNotFound, "user not found")
		}
		if existing, _ := app.FindFirstRecordByFilter("league_members",
			"league = {:l} && user = {:u}",
			map[string]any{"l": leagueID, "u": userID}); existing != nil {
			return bad(e, http.StatusConflict, "user is already a member")
		}
		if pending, _ := app.FindFirstRecordByFilter(leagueInvitesCollection,
			"league = {:l} && invitedUser = {:u} && status = {:s}",
			map[string]any{"l": leagueID, "u": userID, "s": inviteStatusPending}); pending != nil {
			return bad(e, http.StatusConflict, "invite already pending")
		}

		col, err := app.FindCollectionByNameOrId(leagueInvitesCollection)
		if err != nil {
			return err
		}
		invite := core.NewRecord(col)
		invite.Set("league", leagueID)
		invite.Set("invitedUser", userID)
		invite.Set("invitedBy", e.Auth.Id)
		invite.Set("status", inviteStatusPending)
		if err := app.Save(invite); err != nil {
			return err
		}
		item, err := leagueInviteInfo(app, invite, true)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"invite": item})
	})

	// GET /api/leagues/invitations
	g.GET("/invitations", func(e *core.RequestEvent) error {
		recs, err := app.FindRecordsByFilter(leagueInvitesCollection,
			"invitedUser = {:u} && status = {:s}", "-created", 0, 0,
			map[string]any{"u": e.Auth.Id, "s": inviteStatusPending})
		if err != nil {
			return err
		}
		out := make([]leagueInviteDTO, 0, len(recs))
		for _, rec := range recs {
			item, err := leagueInviteInfo(app, rec, false)
			if err != nil || item.LeagueName == "" {
				continue
			}
			out = append(out, item)
		}
		return e.JSON(http.StatusOK, map[string]any{"invites": out})
	})

	// POST /api/leagues/invitations/{inviteId}/accept
	g.POST("/invitations/{inviteId}/accept", func(e *core.RequestEvent) error {
		inviteID := e.Request.PathValue("inviteId")
		var leagueOut map[string]any
		if err := app.RunInTransaction(func(tx core.App) error {
			invite, err := tx.FindRecordById(leagueInvitesCollection, inviteID)
			if err != nil {
				return apis.NewNotFoundError("invite not found", nil)
			}
			if invite.GetString("invitedUser") != e.Auth.Id || invite.GetString("status") != inviteStatusPending {
				return apis.NewForbiddenError("invite not available", nil)
			}
			league, err := tx.FindRecordById("leagues", invite.GetString("league"))
			if err != nil {
				return apis.NewNotFoundError("league not found", nil)
			}
			if league.GetString("inviteCode") == GlobalInviteCode {
				return apis.NewForbiddenError("global league cannot be invited to", nil)
			}
			if existing, _ := tx.FindFirstRecordByFilter("league_members",
				"league = {:l} && user = {:u}",
				map[string]any{"l": league.Id, "u": e.Auth.Id}); existing == nil {
				if err := addMember(tx, league.Id, e.Auth.Id, "member"); err != nil {
					return err
				}
			}
			invite.Set("status", inviteStatusAccepted)
			invite.Set("actedAt", time.Now().UTC())
			if err := tx.Save(invite); err != nil {
				return err
			}
			leagueOut = map[string]any{"id": league.Id, "name": league.GetString("name")}
			return nil
		}); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"league": leagueOut})
	})

	// POST /api/leagues/invitations/{inviteId}/decline
	g.POST("/invitations/{inviteId}/decline", func(e *core.RequestEvent) error {
		inviteID := e.Request.PathValue("inviteId")
		invite, err := app.FindRecordById(leagueInvitesCollection, inviteID)
		if err != nil {
			return bad(e, http.StatusNotFound, "invite not found")
		}
		if invite.GetString("invitedUser") != e.Auth.Id || invite.GetString("status") != inviteStatusPending {
			return bad(e, http.StatusForbidden, "invite not available")
		}
		invite.Set("status", inviteStatusDeclined)
		invite.Set("actedAt", time.Now().UTC())
		if err := app.Save(invite); err != nil {
			return err
		}
		return e.NoContent(http.StatusNoContent)
	})

	// GET /api/leagues/{id}/leaderboard
	g.GET("/{id}/leaderboard", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if _, err := app.FindFirstRecordByFilter("league_members",
			"league = {:l} && user = {:u}",
			map[string]any{"l": id, "u": e.Auth.Id}); err != nil {
			return bad(e, http.StatusForbidden, "not a member of this league")
		}
		lb, err := scoring.Leaderboard(app, id)
		if err != nil {
			return bad(e, http.StatusNotFound, "league not found")
		}
		// Include the league's scoring config so the legend can render it
		// without the client reading the (now members-only) leagues table.
		if lg, err := app.FindRecordById("leagues", id); err == nil {
			cid := lg.GetString("scoringConfig")
			var sc *core.Record
			if cid != "" {
				sc, _ = app.FindRecordById("scoring_configs", cid)
			}
			if sc == nil {
				sc, _ = app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
			}
			if sc != nil {
				var cfg map[string]any
				if json.Unmarshal([]byte(sc.GetString("config")), &cfg) == nil {
					lb["scoring"] = cfg
				}
			}
		}
		return e.JSON(http.StatusOK, lb)
	})

	// GET /api/leagues/{id}/progress
	g.GET("/{id}/progress", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if _, err := app.FindFirstRecordByFilter("league_members",
			"league = {:l} && user = {:u}",
			map[string]any{"l": id, "u": e.Auth.Id}); err != nil {
			return bad(e, http.StatusForbidden, "not a member of this league")
		}
		progress, err := scoring.LeagueProgress(app, id, e.Auth.Id, 16)
		if err != nil {
			return bad(e, http.StatusNotFound, "league not found")
		}
		return e.JSON(http.StatusOK, progress)
	})
}

func addMember(app core.App, leagueID, userID, role string) error {
	col, err := app.FindCollectionByNameOrId("league_members")
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("league", leagueID)
	rec.Set("user", userID)
	rec.Set("role", role)
	return app.Save(rec)
}

func isBotUser(user *core.Record) bool {
	if user == nil {
		return false
	}
	email := strings.ToLower(strings.TrimSpace(user.GetString("email")))
	return strings.HasSuffix(email, botEmailSuffix)
}

func removeGlobalMember(app core.App, leagueID, userID string) error {
	recs, err := app.FindRecordsByFilter("league_members",
		"league = {:l} && user = {:u}", "", 0, 0,
		map[string]any{"l": leagueID, "u": userID})
	if err != nil {
		return err
	}
	for _, rec := range recs {
		if err := app.Delete(rec); err != nil {
			return err
		}
	}
	return nil
}

// ensureGlobal idempotently creates the "Global" league (owner left empty so
// no one can update/delete it via REST). Returns the league id.
func ensureGlobal(app core.App) (string, error) {
	if rec, err := app.FindFirstRecordByFilter("leagues",
		"inviteCode = {:c}", map[string]any{"c": GlobalInviteCode}); err == nil {
		return rec.Id, nil
	}
	col, err := app.FindCollectionByNameOrId("leagues")
	if err != nil {
		return "", err
	}
	def, _ := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
	rec := core.NewRecord(col)
	rec.Set("name", "Global")
	rec.Set("inviteCode", GlobalInviteCode)
	if def != nil {
		rec.Set("scoringConfig", def.Id)
	}
	if err := app.Save(rec); err != nil {
		return "", err
	}
	return rec.Id, nil
}

// ensureGlobalMember adds the user to the Global league if not already a member.
func ensureGlobalMember(app core.App, userID string) error {
	leagueID, err := ensureGlobal(app)
	if err != nil {
		return err
	}
	if existing, _ := app.FindFirstRecordByFilter("league_members",
		"league = {:l} && user = {:u}",
		map[string]any{"l": leagueID, "u": userID}); existing != nil {
		return nil
	}
	return addMember(app, leagueID, userID, "member")
}

// backfillGlobal ensures every existing user is a member of the Global league.
// Cheap on subsequent boots: the per-user membership check short-circuits.
func backfillGlobal(app core.App) error {
	leagueID, err := ensureGlobal(app)
	if err != nil {
		return err
	}
	users, err := app.FindRecordsByFilter("users", "id != ''", "", 0, 0)
	if err != nil {
		return err
	}
	for _, u := range users {
		if isBotUser(u) {
			if err := removeGlobalMember(app, leagueID, u.Id); err != nil {
				return err
			}
			continue
		}
		if err := ensureGlobalMember(app, u.Id); err != nil {
			return err
		}
	}
	return nil
}
