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
	if email == "" || alreadySent(app, dedupKey, ChannelEmail) {
		return false
	}
	out, err := render(app, event, u.GetString("language"), data)
	if err != nil {
		return false
	}
	if err := sendEmail(app, email, out.Subject, out.HTML); err != nil {
		log.Printf("[notifications] email %s -> %s failed: %v", event, email, err)
		return false
	}
	_ = recordSend(app, u.Id, email, event, ChannelEmail, dedupKey, statusSent, "")
	log.Printf("[notifications] email %s sent to %s", event, email)
	return true
}

// deliverPush sends one push notification to all the user's devices with dedup.
func deliverPush(app core.App, u *core.Record, event, dedupKey string, data renderData) bool {
	if !Wants(u, event, ChannelPush) {
		return false
	}
	if alreadySent(app, dedupKey, ChannelPush) {
		return false
	}
	payload, ok := renderPushPayload(app, event, u.GetString("language"), data)
	if !ok {
		return false
	}
	if !sendPushToUser(app, u.Id, payload) {
		return false
	}
	_ = recordSend(app, u.Id, u.GetString("email"), event, ChannelPush, dedupKey, statusSent, "")
	log.Printf("[notifications] push %s sent to user %s", event, u.Id)
	return true
}
