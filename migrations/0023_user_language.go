package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if users.Fields.GetByName("language") == nil {
			users.Fields.Add(&core.SelectField{
				Name:      "language",
				MaxSelect: 1,
				Values:    []string{"en", "nb", "nn"},
			})
		}
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if users.Fields.GetByName("language") != nil {
			users.Fields.RemoveByName("language")
		}
		return app.Save(users)
	})
}