package notifications

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/clock"
)

const sendsCollection = "notification_sends"

// Send-log status values.
const (
	statusSending = "sending"
	statusSent    = "sent"
	statusFailed  = "failed"
	statusSkipped = "skipped"
)

// alreadySent reports whether a successful delivery already exists for this
// dedupKey on this channel. This is the idempotency check that stops a restart
// or a repeated cron tick from sending the same notification twice.
func alreadySent(app core.App, dedupKey, channel string) bool {
	recs, err := app.FindRecordsByFilter(sendsCollection,
		"dedupKey = {:k} && channel = {:c} && status = {:s}",
		"", 1, 0,
		map[string]any{"k": dedupKey, "c": channel, "s": statusSent})
	return err == nil && len(recs) > 0
}

func findSendRecord(app core.App, dedupKey, channel string) (*core.Record, error) {
	recs, err := app.FindRecordsByFilter(sendsCollection,
		"dedupKey = {:k} && channel = {:c}",
		"", 1, 0,
		map[string]any{"k": dedupKey, "c": channel})
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return recs[0], nil
}

// claimSend acquires the dedup slot before delivery. If claimed is false, a
// send is already in-progress or completed for this key/channel and the caller
// must skip sending.
func claimSend(app core.App, userID, email, event, channel, dedupKey string) (rec *core.Record, claimed bool, err error) {
	rec, err = findSendRecord(app, dedupKey, channel)
	if err != nil {
		return nil, false, err
	}
	if rec != nil {
		switch rec.GetString("status") {
		case statusSent, statusSending:
			return rec, false, nil
		default:
			rec.Set("status", statusSending)
			rec.Set("error", "")
			rec.Set("event", event)
			rec.Set("email", email)
			if userID != "" {
				rec.Set("user", userID)
			}
			rec.Set("sentAt", "")
			if err := app.Save(rec); err != nil {
				return nil, false, err
			}
			return rec, true, nil
		}
	}

	col, err := app.FindCollectionByNameOrId(sendsCollection)
	if err != nil {
		return nil, false, err
	}
	rec = core.NewRecord(col)
	if userID != "" {
		rec.Set("user", userID)
	}
	rec.Set("email", email)
	rec.Set("event", event)
	rec.Set("channel", channel)
	rec.Set("dedupKey", dedupKey)
	rec.Set("status", statusSending)

	if err := app.Save(rec); err != nil {
		// Another worker may have claimed it first.
		existing, findErr := findSendRecord(app, dedupKey, channel)
		if findErr == nil && existing != nil {
			return existing, false, nil
		}
		return nil, false, err
	}

	return rec, true, nil
}

// finalizeSend marks the claimed record as sent/failed.
func finalizeSend(app core.App, rec *core.Record, sent bool, errMsg string) error {
	if rec == nil {
		return nil
	}
	if sent {
		rec.Set("status", statusSent)
		rec.Set("error", "")
		rec.Set("sentAt", clock.Now(app))
	} else {
		rec.Set("status", statusFailed)
		rec.Set("sentAt", "")
		rec.Set("error", truncate(errMsg, 500))
	}
	return app.Save(rec)
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
