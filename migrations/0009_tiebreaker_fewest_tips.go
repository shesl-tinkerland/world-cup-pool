package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Insert the new "fewestTips" tiebreaker into every existing scoring_configs
// record, just before "earliestEdit" (or appended if earliestEdit is absent).
// Idempotent — skips rows that already include "fewestTips". New installs get
// the right shape from seed.DefaultScoringConfig.
func init() {
	m.Register(func(app core.App) error {
		recs, err := app.FindAllRecords("scoring_configs")
		if err != nil {
			return nil
		}
		for _, rec := range recs {
			var cfg map[string]any
			if err := json.Unmarshal([]byte(rec.GetString("config")), &cfg); err != nil {
				continue
			}
			raw, ok := cfg["tiebreakers"].([]any)
			if !ok {
				continue
			}
			tbs := make([]string, 0, len(raw)+1)
			for _, v := range raw {
				if s, ok := v.(string); ok {
					tbs = append(tbs, s)
				}
			}
			already := false
			for _, t := range tbs {
				if t == "fewestTips" {
					already = true
					break
				}
			}
			if already {
				continue
			}
			out := make([]string, 0, len(tbs)+1)
			inserted := false
			for _, t := range tbs {
				if t == "earliestEdit" && !inserted {
					out = append(out, "fewestTips")
					inserted = true
				}
				out = append(out, t)
			}
			if !inserted {
				out = append(out, "fewestTips")
			}
			cfg["tiebreakers"] = out
			b, err := json.Marshal(cfg)
			if err != nil {
				continue
			}
			rec.Set("config", string(b))
			if err := app.Save(rec); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		return nil // non-destructive
	})
}
