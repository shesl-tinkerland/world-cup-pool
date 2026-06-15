package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

var matchStatusValues = []string{
	"scheduled",
	"live",
	"1H",
	"HT",
	"2H",
	"ET",
	"BT",
	"P",
	"LIVE",
	"INT",
	"finished",
	"postponed",
	"cancelled",
}

var baseMatchStatusValues = []string{
	"scheduled",
	"live",
	"finished",
	"postponed",
	"cancelled",
}

func init() {
	m.Register(func(app core.App) error {
		matches, err := app.FindCollectionByNameOrId(nMatches)
		if err != nil {
			return err
		}
		if f, ok := matches.Fields.GetByName("status").(*core.SelectField); ok {
			f.Values = matchStatusValues
			return app.Save(matches)
		}
		return nil
	}, func(app core.App) error {
		matches, err := app.FindCollectionByNameOrId(nMatches)
		if err != nil {
			return err
		}
		if f, ok := matches.Fields.GetByName("status").(*core.SelectField); ok {
			f.Values = baseMatchStatusValues
			return app.Save(matches)
		}
		return nil
	})
}