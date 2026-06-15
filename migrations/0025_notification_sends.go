package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const nNotificationSends = "notification_sends"

// 0025 adds the send log. Every delivery attempt (email or push) is recorded
// with a stable dedupKey so a restart, a double cron run, or a manual test can
// never send the same notification to the same user twice. The unique index on
// (dedupKey, channel) is the idempotency guarantee.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Backend-only: written exclusively by the notifications package.
		sends := core.NewBaseCollection(nNotificationSends)
		sends.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true})
		sends.Fields.Add(&core.TextField{Name: "email", Max: 255})
		sends.Fields.Add(&core.TextField{Name: "event", Required: true, Max: 100})
		sends.Fields.Add(&core.TextField{Name: "channel", Required: true, Max: 20})
		sends.Fields.Add(&core.TextField{Name: "dedupKey", Required: true, Max: 255})
		sends.Fields.Add(&core.TextField{Name: "status", Max: 20})
		sends.Fields.Add(&core.TextField{Name: "error", Max: 500})
		sends.Fields.Add(&core.DateField{Name: "sentAt"})
		sends.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		sends.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		sends.AddIndex("idx_notification_sends_dedup", true, "dedupKey, channel", "")
		sends.AddIndex("idx_notification_sends_user", false, "user", "")
		return app.Save(sends)
	}, func(app core.App) error {
		if col, err := app.FindCollectionByNameOrId(nNotificationSends); err == nil {
			return app.Delete(col)
		}
		return nil
	})
}
