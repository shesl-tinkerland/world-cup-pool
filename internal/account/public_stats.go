package account

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const publicStatsBotEmailSuffix = "@dev.local"

type PublicStatsResponse struct {
	Users       PublicUserStats `json:"users"`
	GeneratedAt string          `json:"generatedAt"`
}

type PublicUserStats struct {
	Total    int `json:"total"`
	Verified int `json:"verified"`
}

// RegisterPublicStats wires an opt-in GET /api/public/stats endpoint for
// external monitoring tools such as Home Assistant.
func RegisterPublicStats(app core.App, se *core.ServeEvent) {
	token := strings.TrimSpace(os.Getenv("PUBLIC_STATS_TOKEN"))
	if token == "" {
		return
	}

	se.Router.GET("/api/public/stats", func(e *core.RequestEvent) error {
		if strings.TrimSpace(e.Request.Header.Get("Authorization")) != "Bearer "+token {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		users, err := app.FindRecordsByFilter("users", "id != ''", "", 0, 0)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		stats := PublicStatsResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		}
		for _, user := range users {
			if isSyntheticPublicStatsUser(user) {
				continue
			}
			stats.Users.Total++
			if user.GetBool("verified") {
				stats.Users.Verified++
			}
		}

		return e.JSON(http.StatusOK, stats)
	})
}

func isSyntheticPublicStatsUser(user *core.Record) bool {
	if user == nil {
		return false
	}
	email := strings.ToLower(strings.TrimSpace(user.GetString("email")))
	return strings.HasSuffix(email, publicStatsBotEmailSuffix)
}