package notifications

import (
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/clock"
)

func bad(e *core.RequestEvent, code int, msg string) error {
	return e.JSON(code, map[string]string{"error": msg})
}

// isAppSuperuser reports whether the authenticated user's email matches a
// PocketBase superuser record. This intentionally checks the auth record id,
// not just email equality with a superuser, so a normal user account with the
// same address cannot access admin-only notification routes.
func isAppSuperuser(app core.App, user *core.Record) bool {
	if user == nil || user.Id == "" {
		return false
	}
	email := strings.ToLower(strings.TrimSpace(user.GetString("email")))
	if email == "" {
		return false
	}
	superuser, err := app.FindRecordById("_superusers", user.Id)
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(superuser.GetString("email")), email)
}

// sampleData supplies placeholder dynamic values so admin preview/test renders
// a realistic message for events that depend on runtime data.
func sampleData(event string) renderData {
	if event == EventUpcomingMatchesNotTipped {
		return renderData{UntippedCount: 3}
	}
	return renderData{}
}

// Register wires the notification endpoints:
//
//	GET   /api/account/notify-prefs        – catalog + the caller's current prefs
//	PUT   /api/account/notify-prefs        – replace the caller's prefs (validated)
//	POST  /api/account/notify-prompt-seen  – mark the onboarding popup as seen
//	GET   /api/push/vapid-public-key       – VAPID public key for subscribing
//	POST  /api/push/subscribe              – register a Web Push subscription
//	POST  /api/push/unsubscribe            – remove a subscription by endpoint
//	POST  /api/notifications/preview       – render an email without sending (admin)
//	POST  /api/notifications/test          – send a test email to an address (admin)
//	POST  /api/notifications/send-incomplete – send pre-kickoff email to unfinished users (admin)
//	POST  /api/notifications/run           – run one dispatch pass now (admin)
func Register(app core.App, se *core.ServeEvent) {
	prefs := se.Router.Group("/api/account/notify-prefs")
	prefs.Bind(apis.RequireAuth())

	prefs.GET("", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]any{
			"events":         Catalog,
			"prefs":          ReadPrefs(e.Auth),
			"promptSeen":     e.Auth.GetDateTime("notifyPromptSeenAt").Time().Unix() > 0,
			"pushSubscribed": userHasPushSubscription(app, e.Auth.Id),
		})
	})

	prefs.PUT("", func(e *core.RequestEvent) error {
		var body Prefs
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, "invalid body")
		}
		clean := sanitize(body)
		e.Auth.Set("notifyPrefs", clean)
		if err := app.Save(e.Auth); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"prefs": clean})
	})

	// Mark the one-time onboarding popup as seen (own record only).
	se.Router.POST("/api/account/notify-prompt-seen", func(e *core.RequestEvent) error {
		e.Auth.Set("notifyPromptSeenAt", clock.Now(app))
		if err := app.Save(e.Auth); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	}).Bind(apis.RequireAuth())

	// --- Web Push ---------------------------------------------------------
	push := se.Router.Group("/api/push")
	push.Bind(apis.RequireAuth())

	push.GET("/vapid-public-key", func(e *core.RequestEvent) error {
		key, err := vapidPublicKey(app)
		if err != nil || key == "" {
			return bad(e, http.StatusServiceUnavailable, "push not configured")
		}
		return e.JSON(http.StatusOK, map[string]string{"publicKey": key})
	})

	push.POST("/subscribe", func(e *core.RequestEvent) error {
		var req struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				P256dh string `json:"p256dh"`
				Auth   string `json:"auth"`
			} `json:"keys"`
		}
		if err := e.BindBody(&req); err != nil {
			return bad(e, http.StatusBadRequest, "invalid body")
		}
		if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
			return bad(e, http.StatusBadRequest, "incomplete subscription")
		}
		ua := e.Request.Header.Get("User-Agent")
		if err := saveSubscription(app, e.Auth.Id, req.Endpoint, req.Keys.P256dh, req.Keys.Auth, ua); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})

	// Send a real push to the caller's own registered devices so they can
	// verify the pipeline end to end. Bypasses prefs and the send log — the
	// user asked for it explicitly, and it only ever targets their own devices.
	push.POST("/test", func(e *core.RequestEvent) error {
		subs, err := app.FindRecordsByFilter(pushSubsCollection, "user = {:u}", "", 0, 0, map[string]any{"u": e.Auth.Id})
		if err != nil || len(subs) == 0 {
			return e.JSON(http.StatusOK, map[string]any{"sent": false, "devices": 0})
		}
		ok := sendPushToUser(app, e.Auth.Id, testPushPayload(app, e.Auth.GetString("language")))
		if ok {
			log.Printf("[notifications] test push sent to user %s (%d device(s))", e.Auth.Id, len(subs))
		} else {
			log.Printf("[notifications] test push to user %s failed for all %d device(s)", e.Auth.Id, len(subs))
		}
		return e.JSON(http.StatusOK, map[string]any{"sent": ok, "devices": len(subs)})
	})

	push.POST("/unsubscribe", func(e *core.RequestEvent) error {
		var req struct {
			Endpoint string `json:"endpoint"`
		}
		if err := e.BindBody(&req); err != nil {
			return bad(e, http.StatusBadRequest, "invalid body")
		}
		if req.Endpoint != "" {
			deleteSubscriptionByEndpoint(app, e.Auth.Id, req.Endpoint)
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})

	// --- Admin verification ----------------------------------------------
	admin := se.Router.Group("/api/notifications")
	admin.Bind(apis.RequireAuth())

	admin.POST("/preview", func(e *core.RequestEvent) error {
		if !isAppSuperuser(app, e.Auth) {
			return bad(e, http.StatusForbidden, "admin access required")
		}
		var req struct {
			Event string `json:"event"`
			Lang  string `json:"lang"`
		}
		if err := e.BindBody(&req); err != nil {
			return bad(e, http.StatusBadRequest, "invalid body")
		}
		out, err := render(app, req.Event, req.Lang, sampleData(req.Event))
		if err != nil {
			return bad(e, http.StatusBadRequest, "unknown event")
		}
		return e.JSON(http.StatusOK, out)
	})

	admin.POST("/test", func(e *core.RequestEvent) error {
		if !isAppSuperuser(app, e.Auth) {
			return bad(e, http.StatusForbidden, "admin access required")
		}
		var req struct {
			Event string `json:"event"`
			To    string `json:"to"`
			Lang  string `json:"lang"`
		}
		if err := e.BindBody(&req); err != nil {
			return bad(e, http.StatusBadRequest, "invalid body")
		}
		to := strings.TrimSpace(req.To)
		if to == "" {
			to = e.Auth.GetString("email")
		}
		if _, err := mail.ParseAddress(to); err != nil {
			return bad(e, http.StatusBadRequest, "invalid recipient address")
		}
		out, err := render(app, req.Event, req.Lang, sampleData(req.Event))
		if err != nil {
			return bad(e, http.StatusBadRequest, "unknown event")
		}
		if err := sendEmail(app, to, out.Subject, out.HTML); err != nil {
			log.Printf("[notifications] test send to %s failed: %v", to, err)
			return bad(e, http.StatusBadGateway, "send failed: "+err.Error())
		}
		log.Printf("[notifications] test %q sent to %s", req.Event, to)
		return e.JSON(http.StatusOK, map[string]any{"sent": true, "to": to})
	})

	// Send the pre-kickoff email immediately to every unfinished user. This is
	// a manual admin override for one-off campaigns and intentionally ignores
	// opt-in prefs while reusing the normal email dedup key.
	admin.POST("/send-incomplete", func(e *core.RequestEvent) error {
		if !isAppSuperuser(app, e.Auth) {
			return bad(e, http.StatusForbidden, "admin access required")
		}
		summary, err := SendPreKickoffReminderToIncomplete(app)
		if err != nil {
			return bad(e, http.StatusInternalServerError, err.Error())
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true, "summary": summary})
	})

	// Run one dispatch pass immediately (respects prefs, send log and the
	// pre-kickoff window). Lets an admin verify the cron flow in a test
	// container without waiting for the schedule.
	admin.POST("/run", func(e *core.RequestEvent) error {
		if !isAppSuperuser(app, e.Auth) {
			return bad(e, http.StatusForbidden, "admin access required")
		}
		go RunDispatchNow(app)
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})
}
