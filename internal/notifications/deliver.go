package notifications

import (
	"log"

	"github.com/pocketbase/pocketbase/core"
)

// botSuffix marks local dev-simulation accounts, which never receive notifications.
const botSuffix = "@dev.local"

// deliverEmail renders and sends one email to the user with dedup, recording a
// successful delivery in the send log. It is a no-op (returns false) if the user
// has not opted in, has no address, or was already sent this dedupKey.
func deliverEmail(app core.App, u *core.Record, event, dedupKey string, data renderData) bool {
	if !Wants(u, event, ChannelEmail) {
		return false
	}
	email := u.GetString("email")
	if email == "" {
		return false
	}
	claim, claimed, err := claimSend(app, u.Id, email, event, ChannelEmail, dedupKey)
	if err != nil {
		log.Printf("[notifications] email %s claim failed for %s: %v", event, email, err)
		return false
	}
	if !claimed {
		return false
	}
	out, err := render(app, event, u.GetString("language"), data)
	if err != nil {
		if ferr := finalizeSend(app, claim, false, err.Error()); ferr != nil {
			log.Printf("[notifications] email %s finalize(failed) for %s failed: %v", event, email, ferr)
		}
		return false
	}
	if err := sendEmail(app, email, out.Subject, out.HTML); err != nil {
		log.Printf("[notifications] email %s -> %s failed: %v", event, email, err)
		if ferr := finalizeSend(app, claim, false, err.Error()); ferr != nil {
			log.Printf("[notifications] email %s finalize(failed) for %s failed: %v", event, email, ferr)
		}
		return false
	}
	if err := finalizeSend(app, claim, true, ""); err != nil {
		log.Printf("[notifications] email %s finalize(sent) for %s failed: %v", event, email, err)
	}
	log.Printf("[notifications] email %s sent to %s", event, email)
	return true
}

// deliverPush sends one push notification to all the user's devices with dedup.
func deliverPush(app core.App, u *core.Record, event, dedupKey string, data renderData) bool {
	if !Wants(u, event, ChannelPush) {
		return false
	}
	claim, claimed, err := claimSend(app, u.Id, u.GetString("email"), event, ChannelPush, dedupKey)
	if err != nil {
		log.Printf("[notifications] push %s claim failed for user %s: %v", event, u.Id, err)
		return false
	}
	if !claimed {
		return false
	}
	payload, ok := renderPushPayload(app, event, u.GetString("language"), data)
	if !ok {
		if err := finalizeSend(app, claim, false, "render push payload failed"); err != nil {
			log.Printf("[notifications] push %s finalize(failed) for user %s failed: %v", event, u.Id, err)
		}
		return false
	}
	if !sendPushToUser(app, u.Id, payload) {
		if err := finalizeSend(app, claim, false, "send push failed"); err != nil {
			log.Printf("[notifications] push %s finalize(failed) for user %s failed: %v", event, u.Id, err)
		}
		return false
	}
	if err := finalizeSend(app, claim, true, ""); err != nil {
		log.Printf("[notifications] push %s finalize(sent) for user %s failed: %v", event, u.Id, err)
	}
	log.Printf("[notifications] push %s sent to user %s", event, u.Id)
	return true
}
