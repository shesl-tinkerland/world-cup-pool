package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Tighten Tips visibility: a user may only see their OWN tips through the
// collection API. Other members' tips are exposed exclusively via the custom
// /api/tips/others/{matchId} endpoint, which enforces the "only after kickoff
// and only within a shared League" rule (see internal/tips). The Phase 1
// migration left list/view at "any authed user", which would have leaked
// everyone's picks.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("tips")
		if err != nil {
			return err
		}
		own := "user = @request.auth.id"
		c.ListRule = types.Pointer(own)
		c.ViewRule = types.Pointer(own)
		c.CreateRule = types.Pointer(own)
		c.UpdateRule = types.Pointer(own)
		c.DeleteRule = types.Pointer(own)
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("tips")
		if err != nil {
			return err
		}
		authed := "@request.auth.id != ''"
		c.ListRule = types.Pointer(authed)
		c.ViewRule = types.Pointer(authed)
		return app.Save(c)
	})
}
