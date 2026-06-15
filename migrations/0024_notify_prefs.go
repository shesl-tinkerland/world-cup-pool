package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// 0024 adds a free-form JSON field `notifyPrefs` to users. It maps an event
// key to per-channel opt-in flags, e.g.
//
//	{ "pre_kickoff_reminder": { "email": true, "push": false } }
//
// An empty/missing value means "all off" (opt-in): new notification types and
// new channels (e.g. Web Push) can be added later without a schema migration.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if users.Fields.GetByName("notifyPrefs") == nil {
			users.Fields.Add(&core.JSONField{
				Name:    "notifyPrefs",
				MaxSize: 4096,
			})
		}
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if users.Fields.GetByName("notifyPrefs") != nil {
			users.Fields.RemoveByName("notifyPrefs")
		}
		return app.Save(users)
	})
}
