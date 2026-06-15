package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Make leagues.owner optional so the auto-managed "Global" league can exist
// without a real owner (the collection's UpdateRule/DeleteRule still gate on
// `@request.auth.id = owner`, so an empty owner means no one can mutate it
// through the REST API — only superusers can).
func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("leagues")
		if err != nil {
			return err
		}
		if f, ok := col.Fields.GetByName("owner").(*core.RelationField); ok && f.Required {
			f.Required = false
			if err := app.Save(col); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("leagues")
		if err != nil {
			return err
		}
		if f, ok := col.Fields.GetByName("owner").(*core.RelationField); ok && !f.Required {
			f.Required = true
			return app.Save(col)
		}
		return nil
	})
}
