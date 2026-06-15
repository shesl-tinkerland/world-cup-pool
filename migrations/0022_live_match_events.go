package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const nMatchEvents = "match_events"

func init() {
	m.Register(func(app core.App) error {
		matches, err := app.FindCollectionByNameOrId(nMatches)
		if err != nil {
			return err
		}

		if matches.Fields.GetByName("apiFootballFixtureId") == nil {
			matches.Fields.Add(&core.NumberField{Name: "apiFootballFixtureId", OnlyInt: true})
			if err := app.Save(matches); err != nil {
				return err
			}
		}

		if _, err := app.FindCollectionByNameOrId(nMatchEvents); err == nil {
			return nil
		}

		events := core.NewBaseCollection(nMatchEvents)
		events.ListRule = ptr("@request.auth.id != ''")
		events.ViewRule = ptr("@request.auth.id != ''")
		events.CreateRule = nil
		events.UpdateRule = nil
		events.DeleteRule = nil
		events.Fields.Add(&core.RelationField{
			Name:          "match",
			CollectionId:  matches.Id,
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true,
		})
		events.Fields.Add(&core.TextField{Name: "providerKey", Required: true, Max: 80})
		events.Fields.Add(&core.NumberField{Name: "elapsed", OnlyInt: true, Required: true})
		events.Fields.Add(&core.NumberField{Name: "extra", OnlyInt: true})
		events.Fields.Add(&core.TextField{Name: "type", Required: true, Max: 20})
		events.Fields.Add(&core.TextField{Name: "detail", Max: 100})
		events.Fields.Add(&core.TextField{Name: "player", Max: 120})
		events.Fields.Add(&core.TextField{Name: "assist", Max: 120})
		events.Fields.Add(&core.TextField{Name: "team", Max: 120})
		events.Fields.Add(&core.TextField{Name: "comments", Max: 200})
		events.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		events.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		events.AddIndex("idx_match_events_match_provider", true, "match, providerKey", "")
		events.AddIndex("idx_match_events_match_time", false, "match, elapsed, extra", "")
		return app.Save(events)
	}, func(app core.App) error {
		if events, err := app.FindCollectionByNameOrId(nMatchEvents); err == nil {
			if err := app.Delete(events); err != nil {
				return err
			}
		}
		if matches, err := app.FindCollectionByNameOrId(nMatches); err == nil {
			if matches.Fields.GetByName("apiFootballFixtureId") != nil {
				matches.Fields.RemoveByName("apiFootballFixtureId")
				if err := app.Save(matches); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
