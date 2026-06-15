package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const nGoldenBootPlayers = "golden_boot_players"

func init() {
	m.Register(func(app core.App) error {
		teams, err := app.FindCollectionByNameOrId(nTeams)
		if err != nil {
			return err
		}

		if _, err := app.FindCollectionByNameOrId(nGoldenBootPlayers); err != nil {
			col := core.NewBaseCollection(nGoldenBootPlayers)
			col.ListRule = ptr("@request.auth.id != ''")
			col.ViewRule = ptr("@request.auth.id != ''")
			col.Fields.Add(&core.TextField{Name: "providerKey", Required: true, Max: 64})
			col.Fields.Add(&core.NumberField{Name: "providerId", OnlyInt: true})
			col.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 120})
			col.Fields.Add(&core.RelationField{Name: "team", CollectionId: teams.Id, MaxSelect: 1, Required: true})
			col.Fields.Add(&core.TextField{Name: "photoUrl", Max: 500})
			col.Fields.Add(&core.NumberField{Name: "goals", OnlyInt: true})
			col.Fields.Add(&core.NumberField{Name: "assists", OnlyInt: true})
			col.Fields.Add(&core.NumberField{Name: "rank", OnlyInt: true})
			col.Fields.Add(&core.BoolField{Name: "eligible"})
			col.Fields.Add(&core.BoolField{Name: "seeded"})
			col.Fields.Add(&core.DateField{Name: "syncedAt"})
			col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
			col.AddIndex("idx_golden_boot_provider_key", true, "providerKey", "")
			col.AddIndex("idx_golden_boot_rank", false, "rank", "")
			col.AddIndex("idx_golden_boot_eligible", false, "eligible", "")
			if err := app.Save(col); err != nil {
				return err
			}
		}

		players, err := app.FindCollectionByNameOrId(nGoldenBootPlayers)
		if err != nil {
			return err
		}
		forecasts, err := app.FindCollectionByNameOrId(nForecasts)
		if err != nil {
			return err
		}
		if forecasts.Fields.GetByName("goldenBootPlayer") == nil {
			forecasts.Fields.Add(&core.RelationField{Name: "goldenBootPlayer", CollectionId: players.Id, MaxSelect: 1})
			if err := app.Save(forecasts); err != nil {
				return err
			}
		}

		return patchGoldenBootConfig(app, true)
	}, func(app core.App) error {
		if forecasts, err := app.FindCollectionByNameOrId(nForecasts); err == nil {
			if forecasts.Fields.GetByName("goldenBootPlayer") != nil {
				forecasts.Fields.RemoveByName("goldenBootPlayer")
				if err := app.Save(forecasts); err != nil {
					return err
				}
			}
		}
		if col, err := app.FindCollectionByNameOrId(nGoldenBootPlayers); err == nil {
			if err := app.Delete(col); err != nil {
				return err
			}
		}
		return patchGoldenBootConfig(app, false)
	})
}

func patchGoldenBootConfig(app core.App, add bool) error {
	records, err := app.FindRecordsByFilter(nScoringConfigs, "id != ''", "", 0, 0)
	if err != nil {
		return nil
	}
	for _, rec := range records {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(rec.GetString("config")), &cfg); err != nil {
			continue
		}
		forecast, _ := cfg["forecast"].(map[string]any)
		if forecast == nil {
			forecast = map[string]any{}
			cfg["forecast"] = forecast
		}
		if add {
			if _, ok := forecast["goldenBootWinner"]; !ok {
				forecast["goldenBootWinner"] = 15
			}
		} else {
			delete(forecast, "goldenBootWinner")
		}
		raw, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			continue
		}
		rec.Set("config", string(raw))
		if err := app.Save(rec); err != nil {
			return err
		}
	}
	return nil
}
