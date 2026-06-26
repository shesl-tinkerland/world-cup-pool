package notifications

import (
	"log"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// manualSendSummary reports the outcome of a one-off admin-triggered send.
// It intentionally reuses the normal pre-kickoff dedup key so a later cron
// pass cannot double-send the same reminder to the same user.
type manualSendSummary struct {
	Incomplete  int `json:"incomplete"`
	Sent        int `json:"sent"`
	AlreadySent int `json:"alreadySent"`
	Failed      int `json:"failed"`
}

// SendPreKickoffReminderToIncomplete sends the normal pre-kickoff reminder by
// email to every unfinished user, ignoring opt-in prefs for this explicit
// admin action. Push is not part of the manual force-send path.
func SendPreKickoffReminderToIncomplete(app core.App) (manualSendSummary, error) {
	dispatchMu.Lock()
	defer dispatchMu.Unlock()

	start, err := firstKickoff(app)
	if err != nil {
		return manualSendSummary{}, err
	}

	summary := manualSendSummary{}
	year := itoa(start.Year())
	for _, u := range eligibleUsers(app) {
		if hasSubmittedEverything(app, u.Id) {
			continue
		}
		summary.Incomplete++

		email := strings.TrimSpace(u.GetString("email"))
		dedup := EventPreKickoffReminder + ":" + year + ":" + u.Id
		if email == "" {
			summary.AlreadySent++
			continue
		}

		claim, claimed, err := claimSend(app, u.Id, email, EventPreKickoffReminder, ChannelEmail, dedup)
		if err != nil {
			summary.Failed++
			log.Printf("[notifications] manual pre-kickoff claim for %s failed: %v", email, err)
			continue
		}
		if !claimed {
			summary.AlreadySent++
			continue
		}

		out, err := render(app, EventPreKickoffReminder, u.GetString("language"), renderData{})
		if err != nil {
			summary.Failed++
			log.Printf("[notifications] manual pre-kickoff render for %s failed: %v", email, err)
			if ferr := finalizeSend(app, claim, false, err.Error()); ferr != nil {
				log.Printf("[notifications] manual pre-kickoff finalize(failed) for %s failed: %v", email, ferr)
			}
			continue
		}
		if err := sendEmail(app, email, out.Subject, out.HTML); err != nil {
			summary.Failed++
			log.Printf("[notifications] manual pre-kickoff send to %s failed: %v", email, err)
			if ferr := finalizeSend(app, claim, false, err.Error()); ferr != nil {
				log.Printf("[notifications] manual pre-kickoff finalize(failed) for %s failed: %v", email, ferr)
			}
			continue
		}
		if err := finalizeSend(app, claim, true, ""); err != nil {
			log.Printf("[notifications] manual pre-kickoff finalize(sent) for %s failed: %v", email, err)
		}
		summary.Sent++
		log.Printf("[notifications] manual pre-kickoff email sent to %s", email)
	}

	return summary, nil
}