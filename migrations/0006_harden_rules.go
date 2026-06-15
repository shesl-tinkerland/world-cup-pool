package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Hardening: stop leaking league invite codes and other players' raw score
// records to any authenticated account.
//   - leagues: owner-only at the collection API. Members reach their leagues
//     exclusively through the membership-checked custom endpoints
//     (/api/leagues/mine, /{id}/leaderboard, …); the leaderboard endpoint now
//     also returns the scoring config the legend needs.
//   - match_scores / forecast_scores: own-only. Leaderboards are built
//     server-side (admin context) so this doesn't affect them.
func init() {
	m.Register(func(app core.App) error {
		setRules := func(name, list, view string) error {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			c.ListRule = types.Pointer(list)
			c.ViewRule = types.Pointer(view)
			return app.Save(c)
		}
		owner := "@request.auth.id = owner"
		if err := setRules("leagues", owner, owner); err != nil {
			return err
		}
		own := "user = @request.auth.id"
		if err := setRules("match_scores", own, own); err != nil {
			return err
		}
		return setRules("forecast_scores", own, own)
	}, func(app core.App) error {
		authed := "@request.auth.id != ''"
		for _, n := range []string{"leagues", "match_scores", "forecast_scores"} {
			c, err := app.FindCollectionByNameOrId(n)
			if err != nil {
				return err
			}
			c.ListRule = types.Pointer(authed)
			c.ViewRule = types.Pointer(authed)
			if err := app.Save(c); err != nil {
				return err
			}
		}
		return nil
	})
}
