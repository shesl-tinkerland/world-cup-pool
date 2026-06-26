package account

import (
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/tips"
)

const globalInviteCode = "GLOBAL"

func bad(e *core.RequestEvent, code int, msg string) error {
	return e.JSON(code, map[string]string{"error": msg})
}

// Register wires account-management endpoints for the signed-in user
// and registers background hooks (e.g. signup email alerts).
func Register(app core.App, se *core.ServeEvent) {
	registerEmailNormalization(app)
	registerSignupAlerts(app)
	g := se.Router.Group("/api/account")
	g.Bind(apis.RequireAuth())

	// DELETE /api/account
	// Deletes the current user after first removing any private leagues they
	// own so required owner relations don't block the account deletion.
	g.DELETE("", func(e *core.RequestEvent) error {
		// Deleting the user cascades away their tips, including tips on locked
		// matches; bypass the tips lock hook so the cascade isn't blocked.
		tips.SetBypass(true)
		defer tips.SetBypass(false)

		ownedLeagues, err := app.FindRecordsByFilter(
			"leagues",
			"owner = {:u}",
			"",
			0,
			0,
			map[string]any{"u": e.Auth.Id},
		)
		if err != nil {
			return err
		}

		for _, league := range ownedLeagues {
			if league.GetString("inviteCode") == globalInviteCode {
				return bad(e, http.StatusForbidden, "kan ikkje slette kontoen til eigaren av Global-ligaen")
			}
			if err := app.Delete(league); err != nil {
				return err
			}
		}

		if err := app.Delete(e.Auth); err != nil {
			return err
		}

		return e.NoContent(http.StatusNoContent)
	})
}
