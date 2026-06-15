package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		leagues, err := app.FindCollectionByNameOrId(nLeagues)
		if err != nil {
			return err
		}
		if leagues.Fields.GetByName("privateCode") == nil {
			leagues.Fields.Add(&core.BoolField{Name: "privateCode"})
			return app.Save(leagues)
		}
		return nil
	}, func(app core.App) error {
		leagues, err := app.FindCollectionByNameOrId(nLeagues)
		if err != nil {
			return err
		}
		if leagues.Fields.GetByName("privateCode") != nil {
			leagues.Fields.RemoveByName("privateCode")
			return app.Save(leagues)
		}
		return nil
	})
}