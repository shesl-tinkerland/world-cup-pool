package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/oyvhov/world-cup-pool/internal/seed"
)

// Add the Norwegian broadcast channel label used by the match cards.
func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("matches")
		if err != nil {
			return err
		}
		if col.Fields.GetByName("tvChannel") == nil {
			col.Fields.Add(&core.TextField{Name: "tvChannel", Max: 8})
			if err := app.Save(col); err != nil {
				return err
			}
		}
		return seed.ApplyTVChannels(app)
	}, func(app core.App) error {
		return nil
	})
}
