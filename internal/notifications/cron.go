package notifications

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/clock"
)

const (
	// preKickoffWindow: the pre-kickoff reminder fires within this span before
	// the tournament's first match (1 day before, per product decision).
	preKickoffWindow = 24 * time.Hour
	// upcomingWindow: matches kicking off within this span count as "upcoming"
	// for the not-tipped reminder.
	upcomingWindow = 24 * time.Hour
)

var dispatchMu sync.Mutex

// StartCron registers the dispatch job. It is opt-in via NOTIFY_CRON_ENABLED=1
// so a deploy can ship with automatic sending off and be turned on once the
// manual flow is verified. The job is idempotent: the send log dedups, so
// running every 15 minutes never double-sends.
func StartCron(app core.App) {
	if os.Getenv("NOTIFY_CRON_ENABLED") != "1" {
		log.Printf("[notifications] cron disabled (set NOTIFY_CRON_ENABLED=1 to enable)")
		return
	}
	app.Cron().MustAdd("notifications-dispatch", "*/15 * * * *", func() {
		runDispatch(app)
	})
	log.Printf("[notifications] cron enabled (*/15 * * * *)")
}

// RunDispatchNow runs one dispatch pass immediately. Exposed for the admin
// "run cron now" endpoint used to verify the flow in a test container.
func RunDispatchNow(app core.App) {
	runDispatch(app)
}

func runDispatch(app core.App) {
	dispatchMu.Lock()
	defer dispatchMu.Unlock()

	now := clock.Now(app)
	dispatchPreKickoff(app, now)
	dispatchUpcoming(app, now)
}

// eligibleUsers returns all real users (dev-simulation bots excluded).
func eligibleUsers(app core.App) []*core.Record {
	all, err := app.FindRecordsByFilter("users", "email != ''", "", 0, 0)
	if err != nil {
		return nil
	}
	out := make([]*core.Record, 0, len(all))
	for _, u := range all {
		if strings.HasSuffix(strings.ToLower(u.GetString("email")), botSuffix) {
			continue
		}
		out = append(out, u)
	}
	return out
}

// dispatchPreKickoff sends the one-time reminder to opted-in users who have not
// submitted everything, during the 24h before the first kickoff.
func dispatchPreKickoff(app core.App, now time.Time) {
	start, err := firstKickoff(app)
	if err != nil {
		return
	}
	if now.Before(start.Add(-preKickoffWindow)) || !now.Before(start) {
		return // outside the window (too early, or tournament already started)
	}
	year := itoa(start.Year())
	for _, u := range eligibleUsers(app) {
		wantsEmail := Wants(u, EventPreKickoffReminder, ChannelEmail)
		wantsPush := Wants(u, EventPreKickoffReminder, ChannelPush)
		if !wantsEmail && !wantsPush {
			continue
		}
		if hasSubmittedEverything(app, u.Id) {
			continue
		}
		dedup := EventPreKickoffReminder + ":" + year + ":" + u.Id
		deliverEmail(app, u, EventPreKickoffReminder, dedup, renderData{})
		deliverPush(app, u, EventPreKickoffReminder, dedup, renderData{})
	}
}

// dispatchUpcoming sends the recurring "matches starting soon, not tipped"
// reminder. Deduped per user per calendar day so it can re-send daily while
// matches remain untipped, but never twice in one day.
func dispatchUpcoming(app core.App, now time.Time) {
	day := now.Format("2006-01-02")
	for _, u := range eligibleUsers(app) {
		wantsEmail := Wants(u, EventUpcomingMatchesNotTipped, ChannelEmail)
		wantsPush := Wants(u, EventUpcomingMatchesNotTipped, ChannelPush)
		if !wantsEmail && !wantsPush {
			continue
		}
		n := countUpcomingUntipped(app, u.Id, now, upcomingWindow)
		if n == 0 {
			continue
		}
		dedup := EventUpcomingMatchesNotTipped + ":" + u.Id + ":" + day
		data := renderData{UntippedCount: n}
		deliverEmail(app, u, EventUpcomingMatchesNotTipped, dedup, data)
		deliverPush(app, u, EventUpcomingMatchesNotTipped, dedup, data)
	}
}
