// Package notifications owns the user-facing notification system: the catalog
// of notification types, per-user preferences (the user chooses what they
// receive), and the email sender. Channels other than email (e.g. Web Push)
// plug into the same catalog and preference model later without schema changes.
package notifications

// Channel identifiers stored in notifyPrefs and (later) the send log.
const (
	ChannelEmail = "email"
	ChannelPush  = "push"
)

// Event keys. Stable identifiers; never rename once shipped — they are stored
// in users.notifyPrefs and used as dedup keys for the send log.
const (
	// EventPreKickoffReminder fires once, ~2 days before the tournament's first
	// kickoff, to users who have not submitted everything (group tips, forecast
	// and golden boot pick).
	EventPreKickoffReminder = "pre_kickoff_reminder"
	// EventUpcomingMatchesNotTipped is a recurring reminder: matches kicking off
	// within the next day that the user has not tipped yet.
	EventUpcomingMatchesNotTipped = "upcoming_matches_not_tipped"
)

// Event describes one notification type the user can opt into.
type Event struct {
	// Key is the stable identifier used in notifyPrefs and the send log.
	Key string `json:"key"`
	// Channels are the delivery channels currently available for this event.
	// Email ships first; push is appended when that channel lands.
	Channels []string `json:"channels"`
}

// Catalog is the ordered list of notification types exposed to users. New
// types are added here; no migration is needed because notifyPrefs is free-form
// JSON. Everything is opt-in: a user receives nothing until they enable it.
var Catalog = []Event{
	{Key: EventPreKickoffReminder, Channels: []string{ChannelEmail, ChannelPush}},
	{Key: EventUpcomingMatchesNotTipped, Channels: []string{ChannelEmail, ChannelPush}},
}

// eventByKey returns the catalog entry for key, if it exists.
func eventByKey(key string) (Event, bool) {
	for _, e := range Catalog {
		if e.Key == key {
			return e, true
		}
	}
	return Event{}, false
}

// supportsChannel reports whether the catalog exposes channel for event.
func (e Event) supportsChannel(channel string) bool {
	for _, c := range e.Channels {
		if c == channel {
			return true
		}
	}
	return false
}
