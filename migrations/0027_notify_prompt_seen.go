package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// 0027 adds notifyPromptSeenAt to users. It records when the user has seen the
// one-time onboarding popup that offers to turn on email/push notifications, so
// the popup is shown exactly once across all the user's devices (server-side
// state, not per-browser localStorage). Empty = not seen yet.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if users.Fields.GetByName("notifyPromptSeenAt") == nil {
			users.Fields.Add(&core.DateField{Name: "notifyPromptSeenAt"})
		}
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if users.Fields.GetByName("notifyPromptSeenAt") != nil {
			users.Fields.RemoveByName("notifyPromptSeenAt")
		}
		return app.Save(users)
	})
}
