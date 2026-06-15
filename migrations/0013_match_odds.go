package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const nMatchOdds = "match_odds"

func init() {
	m.Register(func(app core.App) error {
		matches, err := app.FindCollectionByNameOrId(nMatches)
		if err != nil {
			return err
		}

		col := core.NewBaseCollection(nMatchOdds)
		col.ListRule = ptr("")
		col.ViewRule = ptr("")
		col.Fields.Add(&core.RelationField{
			Name:          "match",
			CollectionId:  matches.Id,
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true,
		})
		col.Fields.Add(&core.NumberField{Name: "pHome"})
		col.Fields.Add(&core.NumberField{Name: "pDraw"})
		col.Fields.Add(&core.NumberField{Name: "pAway"})
		col.Fields.Add(&core.NumberField{Name: "homeOdds"})
		col.Fields.Add(&core.NumberField{Name: "drawOdds"})
		col.Fields.Add(&core.NumberField{Name: "awayOdds"})
		col.Fields.Add(&core.DateField{Name: "syncedAt"})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		col.AddIndex("idx_match_odds_match", true, "match", "")
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId(nMatchOdds)
		if err == nil {
			return app.Delete(col)
		}
		return nil
	})
}
