package notifications

import "github.com/pocketbase/pocketbase/core"

// Prefs maps an event key to per-channel opt-in flags, e.g.
// {"pre_kickoff_reminder": {"email": true}}. It is stored verbatim in
// users.notifyPrefs.
type Prefs map[string]map[string]bool

// ReadPrefs returns the user's stored preferences, never nil.
func ReadPrefs(user *core.Record) Prefs {
	p := Prefs{}
	if user == nil {
		return p
	}
	if err := user.UnmarshalJSONField("notifyPrefs", &p); err != nil || p == nil {
		return Prefs{}
	}
	return p
}

// Wants reports whether the user has opted into event on channel. Absence means
// off (opt-in), so a user with no stored prefs receives nothing.
func Wants(user *core.Record, event, channel string) bool {
	ch, ok := ReadPrefs(user)[event]
	if !ok {
		return false
	}
	return ch[channel]
}

// sanitize drops any event keys or channels not present in the catalog so the
// stored value can never contain arbitrary client-supplied keys.
func sanitize(in Prefs) Prefs {
	out := Prefs{}
	for key, channels := range in {
		ev, ok := eventByKey(key)
		if !ok {
			continue
		}
		clean := map[string]bool{}
		for ch, enabled := range channels {
			if ev.supportsChannel(ch) {
				clean[ch] = enabled
			}
		}
		if len(clean) > 0 {
			out[key] = clean
		}
	}
	return out
}
