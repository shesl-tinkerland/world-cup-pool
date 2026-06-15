package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		forecasts, err := app.FindCollectionByNameOrId(nForecasts)
		if err != nil {
			return err
		}
		if forecasts.Fields.GetByName("goldenBootPicks") == nil {
			forecasts.Fields.Add(&core.JSONField{Name: "goldenBootPicks", MaxSize: 1000})
			if err := app.Save(forecasts); err != nil {
				return err
			}
		}

		records, _ := app.FindRecordsByFilter(nForecasts, "goldenBootPlayer != ''", "", 0, 0)
		for _, record := range records {
			var picks []string
			_ = record.UnmarshalJSONField("goldenBootPicks", &picks)
			if len(picks) == 0 {
				record.Set("goldenBootPicks", []string{record.GetString("goldenBootPlayer")})
				if err := app.Save(record); err != nil {
					return err
				}
			}
		}

		return patchGoldenBootPickConfig(app, true)
	}, func(app core.App) error {
		if forecasts, err := app.FindCollectionByNameOrId(nForecasts); err == nil {
			if forecasts.Fields.GetByName("goldenBootPicks") != nil {
				forecasts.Fields.RemoveByName("goldenBootPicks")
				if err := app.Save(forecasts); err != nil {
					return err
				}
			}
		}
		return patchGoldenBootPickConfig(app, false)
	})
}

func patchGoldenBootPickConfig(app core.App, add bool) error {
	records, err := app.FindRecordsByFilter(nScoringConfigs, "id != ''", "", 0, 0)
	if err != nil {
		return nil
	}
	for _, record := range records {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(record.GetString("config")), &cfg); err != nil {
			continue
		}
		forecast, _ := cfg["forecast"].(map[string]any)
		if forecast == nil {
			forecast = map[string]any{}
			cfg["forecast"] = forecast
		}
		if add {
			forecast["goldenBoot"] = map[string]any{
				"exact":     map[string]any{"1": 15, "2": 10, "3": 6},
				"podiumAny": 3,
			}
		} else {
			delete(forecast, "goldenBoot")
		}
		raw, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			continue
		}
		record.Set("config", string(raw))
		if err := app.Save(record); err != nil {
			return err
		}
	}
	return nil
}
