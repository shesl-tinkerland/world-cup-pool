package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	nPushSubscriptions = "push_subscriptions"
	nPushConfig        = "push_config"
)

// 0016 adds Web Push: per-user push subscriptions, a singleton VAPID key store,
// and the per-user notification preferences toggled from Settings.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Subscriptions are managed exclusively through the /api/push endpoints
		// (which run as the backend), so the collection itself stays locked down.
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
		if err := app.Save(subs); err != nil {
			return err
		}

		// Singleton VAPID key store (used only when env VAPID keys are absent).
		// Backend-only access.
		cfg := core.NewBaseCollection(nPushConfig)
		cfg.Fields.Add(&core.TextField{Name: "publicKey", Max: 255})
		cfg.Fields.Add(&core.TextField{Name: "privateKey", Max: 255})
		cfg.Fields.Add(&core.TextField{Name: "subject", Max: 255})
		cfg.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		cfg.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		if err := app.Save(cfg); err != nil {
			return err
		}

		// Per-user notification preferences. Default true so that opting in
		// (granting permission + subscribing) turns both reminders on; users
		// switch them off individually in Settings. lastNotifiedAt enforces the
		// max-one-push-per-day rule.
		if users.Fields.GetByName("notifyTips") == nil {
			users.Fields.Add(&core.BoolField{Name: "notifyTips"})
		}
		if users.Fields.GetByName("notifyForecast") == nil {
			users.Fields.Add(&core.BoolField{Name: "notifyForecast"})
		}
		if users.Fields.GetByName("pushLastNotifiedAt") == nil {
			users.Fields.Add(&core.DateField{Name: "pushLastNotifiedAt"})
		}
		return app.Save(users)
	}, func(app core.App) error {
		for _, name := range []string{nPushSubscriptions, nPushConfig} {
			if col, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(col); err != nil {
					return err
				}
			}
		}
		if users, err := app.FindCollectionByNameOrId("users"); err == nil {
			for _, f := range []string{"notifyTips", "notifyForecast", "pushLastNotifiedAt"} {
				if fld := users.Fields.GetByName(f); fld != nil {
					users.Fields.RemoveByName(f)
				}
			}
			return app.Save(users)
		}
		return nil
	})
}
