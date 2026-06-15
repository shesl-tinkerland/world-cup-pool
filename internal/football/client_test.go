package football

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFixturesForSeasonParsesAPIResponse(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fixtures" {
			t.Fatalf("path = %s, want /fixtures", r.URL.Path)
		}
		if got := r.URL.Query().Get("league"); got != "1" {
			t.Fatalf("league query = %q, want 1", got)
		}
		if got := r.URL.Query().Get("season"); got != "2022" {
			t.Fatalf("season query = %q, want 2022", got)
		}
		if got := r.Header.Get("x-apisports-key"); got != "secret-key" {
			t.Fatalf("x-apisports-key = %q, want secret-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"errors": {},
			"results": 1,
			"response": [{
				"fixture": { "id": 42, "date": "2022-12-18T15:00:00Z", "status": { "short": "PEN" } },
				"league": { "round": "Final" },
				"teams": { "home": { "name": "Argentina" }, "away": { "name": "France" } },
				"goals": { "home": 3, "away": 3 },
				"score": {
					"fulltime": { "home": 2, "away": 2 },
					"extratime": { "home": 1, "away": 1 },
					"penalty": { "home": 4, "away": 2 }
				}
			}]
		}`))
	}))
	defer server.Close()
	baseURL = server.URL

	fixtures, err := New("secret-key").FixturesForSeason(context.Background(), 2022)
	if err != nil {
		t.Fatalf("FixturesForSeason() error = %v", err)
	}
	if len(fixtures) != 1 {
		t.Fatalf("len(fixtures) = %d, want 1", len(fixtures))
	}
	fixture := fixtures[0]
	if fixture.ID != 42 || fixture.Round != "Final" || fixture.Status != "PEN" {
		t.Fatalf("unexpected fixture metadata: %+v", fixture)
	}
	if fixture.HomeName != "Argentina" || fixture.AwayName != "France" {
		t.Fatalf("teams = %s/%s", fixture.HomeName, fixture.AwayName)
	}
	if !fixture.Finished() || fixture.Live() {
		t.Fatalf("status helpers wrong for %q", fixture.Status)
	}
	if *fixture.FTHome != 2 || *fixture.FTAway != 2 || *fixture.ETHome != 1 || *fixture.ETAway != 1 || *fixture.PenHome != 4 || *fixture.PenAway != 2 {
		t.Fatalf("scores parsed wrong: %+v", fixture)
	}
}

func TestFixturesForSeasonReportsAPIErrors(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":{"plan":"season not available"},"results":0,"response":[]}`))
	}))
	defer server.Close()
	baseURL = server.URL

	if _, err := New("secret-key").FixturesForSeason(context.Background(), 2026); err == nil {
		t.Fatal("FixturesForSeason() error = nil, want API error")
	}
}

func TestFixturesForSeasonUsesLiveGoalsWhenFulltimeScoreMissing(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"errors": {},
			"results": 1,
			"response": [{
				"fixture": { "id": 99, "date": "2026-06-11T19:00:00Z", "status": { "short": "1H" } },
				"league": { "round": "Matchday 1" },
				"teams": { "home": { "name": "Mexico" }, "away": { "name": "South Africa" } },
				"goals": { "home": 1, "away": 0 },
				"score": {
					"fulltime": { "home": null, "away": null },
					"extratime": { "home": null, "away": null },
					"penalty": { "home": null, "away": null }
				}
			}]
		}`))
	}))
	defer server.Close()
	baseURL = server.URL

	fixtures, err := New("secret-key").FixturesForSeason(context.Background(), 2026)
	if err != nil {
		t.Fatalf("FixturesForSeason() error = %v", err)
	}
	if len(fixtures) != 1 {
		t.Fatalf("len(fixtures) = %d, want 1", len(fixtures))
	}
	if fixtures[0].FTHome == nil || fixtures[0].FTAway == nil {
		t.Fatal("live fixture should expose current score as regulation score")
	}
	if *fixtures[0].FTHome != 1 || *fixtures[0].FTAway != 0 {
		t.Fatalf("live score = %v-%v, want 1-0", *fixtures[0].FTHome, *fixtures[0].FTAway)
	}
}

func TestFixtureEventsParsesAPIResponse(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fixtures/events" {
			t.Fatalf("path = %s, want /fixtures/events", r.URL.Path)
		}
		if got := r.URL.Query().Get("fixture"); got != "42" {
			t.Fatalf("fixture query = %q, want 42", got)
		}
		if got := r.Header.Get("x-apisports-key"); got != "secret-key" {
			t.Fatalf("x-apisports-key = %q, want secret-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"errors": {},
			"results": 2,
			"response": [
				{
					"time": { "elapsed": 90, "extra": 4 },
					"team": { "id": 26, "name": "Argentina" },
					"player": { "id": 278, "name": "Lionel Messi" },
					"assist": { "id": 999, "name": "Angel Di Maria" },
					"type": "Goal",
					"detail": "Penalty",
					"comments": "Shootout pressure"
				},
				{
					"time": { "elapsed": 33, "extra": null },
					"team": { "id": 2, "name": "France" },
					"player": { "id": 123, "name": "Adrien Rabiot" },
					"assist": { "id": null, "name": null },
					"type": "Card",
					"detail": "Yellow Card",
					"comments": null
				}
			]
		}`))
	}))
	defer server.Close()
	baseURL = server.URL

	events, err := New("secret-key").FixtureEvents(context.Background(), 42)
	if err != nil {
		t.Fatalf("FixtureEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	first := events[0]
	if first.Elapsed != 90 || first.Extra != 4 || first.TeamID != 26 || first.PlayerID != 278 || first.AssistID != 999 {
		t.Fatalf("event metadata parsed wrong: %+v", first)
	}
	if first.Team != "Argentina" || first.Player != "Lionel Messi" || first.Assist != "Angel Di Maria" || first.Type != "Goal" || first.Detail != "Penalty" || first.Comments != "Shootout pressure" {
		t.Fatalf("event strings parsed wrong: %+v", first)
	}
	second := events[1]
	if second.Extra != 0 || second.AssistID != 0 || second.Assist != "" || second.Comments != "" {
		t.Fatalf("null fields parsed wrong: %+v", second)
	}
}

func TestFixtureEventsNoContentReturnsEmptySlice(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	baseURL = server.URL

	events, err := New("secret-key").FixtureEvents(context.Background(), 42)
	if err != nil {
		t.Fatalf("FixtureEvents() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0", len(events))
	}
}

func TestFixtureEventsReportsAPIErrors(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":{"fixture":"not found"},"results":0,"response":[]}`))
	}))
	defer server.Close()
	baseURL = server.URL

	if _, err := New("secret-key").FixtureEvents(context.Background(), 42); err == nil {
		t.Fatal("FixtureEvents() error = nil, want API error")
	}
}

func TestTopScorersParsesAPIResponse(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/players/topscorers" {
			t.Fatalf("path = %s, want /players/topscorers", r.URL.Path)
		}
		if got := r.URL.Query().Get("league"); got != "1" {
			t.Fatalf("league query = %q, want 1", got)
		}
		if got := r.URL.Query().Get("season"); got != "2026" {
			t.Fatalf("season query = %q, want 2026", got)
		}
		if got := r.Header.Get("x-apisports-key"); got != "secret-key" {
			t.Fatalf("x-apisports-key = %q, want secret-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"errors": {},
			"results": 1,
			"response": [{
				"player": { "id": 278, "name": "Kylian Mbappé", "photo": "https://media.example/mbappe.png" },
				"statistics": [{
					"team": { "name": "France" },
					"goals": { "total": 8, "assists": 2 }
				}]
			}]
		}`))
	}))
	defer server.Close()
	baseURL = server.URL

	scorers, err := New("secret-key").TopScorers(context.Background())
	if err != nil {
		t.Fatalf("TopScorers() error = %v", err)
	}
	if len(scorers) != 1 {
		t.Fatalf("len(scorers) = %d, want 1", len(scorers))
	}
	scorer := scorers[0]
	if scorer.ProviderID != 278 || scorer.Name != "Kylian Mbappé" || scorer.TeamName != "France" {
		t.Fatalf("unexpected scorer metadata: %+v", scorer)
	}
	if scorer.PhotoURL != "https://media.example/mbappe.png" || scorer.Goals != 8 || scorer.Assists != 2 || scorer.Rank != 1 {
		t.Fatalf("unexpected scorer stats: %+v", scorer)
	}
}

func TestSearchPlayersFallsBackToGenericProfile(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-apisports-key"); got != "secret-key" {
			t.Fatalf("x-apisports-key = %q, want secret-key", got)
		}
		call++
		w.Header().Set("Content-Type", "application/json")
		switch call {
		case 1:
			if r.URL.Path != "/players" {
				t.Fatalf("path = %s, want /players", r.URL.Path)
			}
			if got := r.URL.Query().Get("search"); got != "Julian" {
				t.Fatalf("search query = %q, want Julian", got)
			}
			if got := r.URL.Query().Get("league"); got != "1" {
				t.Fatalf("league query = %q, want 1", got)
			}
			if got := r.URL.Query().Get("season"); got != "2026" {
				t.Fatalf("season query = %q, want 2026", got)
			}
			_, _ = w.Write([]byte(`{"errors": {}, "results": 0, "response": []}`))
		case 2:
			if r.URL.Path != "/players/profiles" {
				t.Fatalf("path = %s, want /players/profiles", r.URL.Path)
			}
			if got := r.URL.Query().Get("search"); got != "Julian" {
				t.Fatalf("search query = %q, want Julian", got)
			}
			_, _ = w.Write([]byte(`{
				"errors": {},
				"results": 1,
				"response": [{
					"player": {
						"id": 999,
						"name": "Julian Alvarez",
						"photo": "https://media.example/alvarez.png",
						"nationality": "Argentina"
					},
					"statistics": []
				}]
			}`))
		default:
			t.Fatalf("unexpected call %d", call)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	players, err := New("secret-key").SearchPlayers(context.Background(), "Julian")
	if err != nil {
		t.Fatalf("SearchPlayers() error = %v", err)
	}
	if call != 2 {
		t.Fatalf("call count = %d, want 2", call)
	}
	if len(players) != 1 {
		t.Fatalf("len(players) = %d, want 1", len(players))
	}
	player := players[0]
	if player.ProviderID != 999 || player.Name != "Julian Alvarez" || player.TeamName != "Argentina" {
		t.Fatalf("unexpected player metadata: %+v", player)
	}
	if player.PhotoURL != "https://media.example/alvarez.png" || player.Goals != 0 || player.Assists != 0 {
		t.Fatalf("unexpected player stats: %+v", player)
	}
}

func TestSearchPlayersProfilesPrioritizeExactWordMatches(t *testing.T) {
	oldBaseURL := baseURL
	defer func() { baseURL = oldBaseURL }()

	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		w.Header().Set("Content-Type", "application/json")
		switch call {
		case 1:
			_, _ = w.Write([]byte(`{"errors": {}, "results": 0, "response": []}`))
		case 2:
			if r.URL.Path != "/players/profiles" {
				t.Fatalf("path = %s, want /players/profiles", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{
				"errors": {},
				"results": 3,
				"response": [
					{"player": {"id": 10, "name": "Marcelo Messias", "photo": "https://media.example/1.png", "nationality": "Brazil"}},
					{"player": {"id": 11, "name": "Messi Lionel", "photo": "https://media.example/2.png", "nationality": "Argentina"}},
					{"player": {"id": 12, "name": "Lionel Messi", "photo": "https://media.example/3.png", "nationality": "Argentina"}}
				]
			}`))
		default:
			t.Fatalf("unexpected call %d", call)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	players, err := New("secret-key").SearchPlayers(context.Background(), "Messi")
	if err != nil {
		t.Fatalf("SearchPlayers() error = %v", err)
	}
	if len(players) != 3 {
		t.Fatalf("len(players) = %d, want 3", len(players))
	}
	if players[0].Name != "Lionel Messi" {
		t.Fatalf("first player = %q, want Lionel Messi", players[0].Name)
	}
	if players[1].Name != "Messi Lionel" {
		t.Fatalf("second player = %q, want Messi Lionel", players[1].Name)
	}
}
