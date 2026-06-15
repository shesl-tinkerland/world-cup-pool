package notifications

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/clock"
)

const sendsCollection = "notification_sends"

// Send-log status values.
const (
	statusSent    = "sent"
	statusFailed  = "failed"
	statusSkipped = "skipped"
)

// alreadySent reports whether a successful delivery already exists for this
// dedupKey on this channel. This is the idempotency check that stops a restart
// or a repeated cron tick from sending the same notification twice.
func alreadySent(app core.App, dedupKey, channel string) bool {
	_, err := app.FindFirstRecordByFilter(sendsCollection,
		"dedupKey = {:k} && channel = {:c} && status = {:s}",
		map[string]any{"k": dedupKey, "c": channel, "s": statusSent})
	return err == nil
}

// recordSend appends a row to the send log. The unique (dedupKey, channel)
// index makes a duplicate insert fail rather than double-send. Only successful
// deliveries are recorded, so a transient failure is retried on the next tick.
func recordSend(app core.App, userID, email, event, channel, dedupKey, status, errMsg string) error {
	col, err := app.FindCollectionByNameOrId(sendsCollection)
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	if userID != "" {
		rec.Set("user", userID)
	}
	rec.Set("email", email)
	rec.Set("event", event)
	rec.Set("channel", channel)
	rec.Set("dedupKey", dedupKey)
	rec.Set("status", status)
	if errMsg != "" {
		rec.Set("error", truncate(errMsg, 500))
	}
	if status == statusSent {
		rec.Set("sentAt", clock.Now(app))
	}
	return app.Save(rec)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
