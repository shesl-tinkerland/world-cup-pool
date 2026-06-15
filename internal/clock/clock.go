// Package clock provides the app's notion of "now". Normally it's the real
// time; the dev tool can override it (persisted in app_meta) so locks,
// kickoff-gated visibility and the Forecast deadline behave as on a simulated
// date. Production never sets the override.
package clock

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const metaKey = "dev_clock"

type stored struct {
	T string `json:"t"`
}

// Now returns the effective current time (simulated if a dev override is set).
func Now(app core.App) time.Time {
	if t, ok := Sim(app); ok {
		return t
	}
	return time.Now().UTC()
}

// Sim returns the simulated time and true if a dev override is active.
func Sim(app core.App) (time.Time, bool) {
	rec, err := app.FindFirstRecordByFilter("app_meta",
		"key = {:k}", map[string]any{"k": metaKey})
	if err != nil {
		return time.Time{}, false
	}
	var v stored
	if err := rec.UnmarshalJSONField("value", &v); err != nil || v.T == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, v.T)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// Set pins the simulated time.
func Set(app core.App, t time.Time) error {
	col, err := app.FindCollectionByNameOrId("app_meta")
	if err != nil {
		return err
	}
	rec, err := app.FindFirstRecordByFilter("app_meta",
		"key = {:k}", map[string]any{"k": metaKey})
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("key", metaKey)
	}
	rec.Set("value", map[string]any{"t": t.UTC().Format(time.RFC3339Nano)})
	return app.Save(rec)
}

// Clear removes the override (back to real time).
func Clear(app core.App) error {
	rec, err := app.FindFirstRecordByFilter("app_meta",
		"key = {:k}", map[string]any{"k": metaKey})
	if err != nil {
		return nil
	}
	return app.Delete(rec)
}
