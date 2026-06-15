package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// nPushSubscriptions is declared in 0016 (which 0017 reverted); reuse it.
//
// 0026 (re)adds Web Push subscriptions. A user can have several (one per
// device/browser). Rows are managed exclusively through the /api/push endpoints
// which run as the backend, so the collection itself stays locked down with no
// API rules. VAPID keys live in app_meta (see internal/notifications/push.go),
// not in a collection, so a backup/restore carries them automatically.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		subs := core.NewBaseCollection(nPushSubscriptions)
		subs.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		subs.Fields.Add(&core.TextField{Name: "endpoint", Required: true, Max: 1000})
		subs.Fields.Add(&core.TextField{Name: "p256dh", Required: true, Max: 255})
		subs.Fields.Add(&core.TextField{Name: "auth", Required: true, Max: 255})
		subs.Fields.Add(&core.TextField{Name: "userAgent", Max: 400})
		subs.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		subs.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		subs.AddIndex("idx_push_subscriptions_endpoint", true, "endpoint", "")
		subs.AddIndex("idx_push_subscriptions_user", false, "user", "")
		return app.Save(subs)
	}, func(app core.App) error {
		if col, err := app.FindCollectionByNameOrId(nPushSubscriptions); err == nil {
			return app.Delete(col)
		}
		return nil
	})
}
