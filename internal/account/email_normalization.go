package account

import (
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerEmailNormalization(app core.App) {
	app.OnRecordCreate("users").BindFunc(func(e *core.RecordEvent) error {
		if err := normalizeNewUserEmail(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})
}

func normalizeNewUserEmail(app core.App, record *core.Record) error {
	email := normalizeEmailAddress(record.GetString("email"))
	if email == "" {
		return nil
	}

	if existing, err := findUserByNormalizedEmail(app, email, record.Id); err != nil {
		return err
	} else if existing != nil {
		return apis.NewBadRequestError("email already registered", nil)
	}

	record.SetEmail(email)
	return nil
}

func normalizeEmailAddress(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func findUserByNormalizedEmail(app core.App, normalizedEmail, excludeID string) (*core.Record, error) {
	users, err := app.FindRecordsByFilter("users", "email != ''", "", 0, 0)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user.Id == excludeID {
			continue
		}
		if normalizeEmailAddress(user.GetString("email")) == normalizedEmail {
			return user, nil
		}
	}
	return nil, nil
}
