package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/oyvhov/world-cup-pool/internal/seed"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId(nTeams)
		if err != nil {
			return err
		}
		if col.Fields.GetByName("fifaRanking") == nil {
			col.Fields.Add(&core.NumberField{Name: "fifaRanking", OnlyInt: true})
			if err := app.Save(col); err != nil {
				return err
			}
		}
		return seed.ApplyFIFARankings(app)
	}, func(app core.App) error {
		return nil
	})
}
