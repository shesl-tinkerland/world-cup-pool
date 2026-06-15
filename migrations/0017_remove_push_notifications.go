package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	nPushSubscriptionsRemoved = "push_subscriptions"
	nPushConfigRemoved        = "push_config"
)

// 0017 removes the Web Push notification feature and its persisted schema.
func init() {
	m.Register(func(app core.App) error {
		for _, name := range []string{nPushSubscriptionsRemoved, nPushConfigRemoved} {
			if col, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(col); err != nil {
					return err
				}
			}
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}
		for _, field := range []string{"notifyTips", "notifyForecast", "pushLastNotifiedAt"} {
			if users.Fields.GetByName(field) != nil {
				users.Fields.RemoveByName(field)
			}
		}
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if users.Fields.GetByName("notifyTips") == nil {
			users.Fields.Add(&core.BoolField{Name: "notifyTips"})
		}
		if users.Fields.GetByName("notifyForecast") == nil {
			users.Fields.Add(&core.BoolField{Name: "notifyForecast"})
		}
		if users.Fields.GetByName("pushLastNotifiedAt") == nil {
			users.Fields.Add(&core.DateField{Name: "pushLastNotifiedAt"})
		}
		if err := app.Save(users); err != nil {
			return err
		}

		if _, err := app.FindCollectionByNameOrId(nPushSubscriptionsRemoved); err != nil {
			subs := core.NewBaseCollection(nPushSubscriptionsRemoved)
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
		}

		if _, err := app.FindCollectionByNameOrId(nPushConfigRemoved); err != nil {
			cfg := core.NewBaseCollection(nPushConfigRemoved)
			cfg.Fields.Add(&core.TextField{Name: "publicKey", Max: 255})
			cfg.Fields.Add(&core.TextField{Name: "privateKey", Max: 255})
			cfg.Fields.Add(&core.TextField{Name: "subject", Max: 255})
			cfg.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			cfg.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
			if err := app.Save(cfg); err != nil {
				return err
			}
		}

		return nil
	})
}