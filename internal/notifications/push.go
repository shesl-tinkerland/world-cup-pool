package notifications

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/pocketbase/pocketbase/core"
)

const (
	pushSubsCollection = "push_subscriptions"
	vapidMetaKey       = "vapid"
)

// vapidKeys holds the application's VAPID keypair and subscriber contact.
type vapidKeys struct {
	Public  string `json:"public"`
	Private string `json:"private"`
}

// vapidSubject is the "Subscriber" contact required by the Web Push spec.
func vapidSubject(app core.App) string {
	if s := strings.TrimSpace(os.Getenv("PUSH_SUBJECT")); s != "" {
		return s
	}
	if addr := strings.TrimSpace(app.Settings().Meta.SenderAddress); addr != "" {
		return "mailto:" + addr
	}
	return "mailto:admin@worldcup.local"
}

// vapid returns the active VAPID keys. Env vars win (set them in production so
// keys are stable and independent of the database); otherwise a pair is
// generated once and persisted in app_meta, so it survives restarts and is
// carried along by a backup/restore.
func vapid(app core.App) (vapidKeys, error) {
	if pub := strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY")); pub != "" {
		return vapidKeys{Public: pub, Private: strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY"))}, nil
	}
	if k, ok := readVapidMeta(app); ok {
		return k, nil
	}
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return vapidKeys{}, err
	}
	k := vapidKeys{Public: pub, Private: priv}
	if err := writeVapidMeta(app, k); err != nil {
		return vapidKeys{}, err
	}
	log.Printf("[notifications] generated VAPID keypair (stored in app_meta)")
	return k, nil
}

// vapidPublicKey returns just the public key for the frontend to subscribe with.
func vapidPublicKey(app core.App) (string, error) {
	k, err := vapid(app)
	if err != nil {
		return "", err
	}
	return k.Public, nil
}

func readVapidMeta(app core.App) (vapidKeys, bool) {
	rec, err := app.FindFirstRecordByFilter("app_meta", "key = {:k}", map[string]any{"k": vapidMetaKey})
	if err != nil {
		return vapidKeys{}, false
	}
	var k vapidKeys
	if err := rec.UnmarshalJSONField("value", &k); err != nil || k.Public == "" || k.Private == "" {
		return vapidKeys{}, false
	}
	return k, true
}

func writeVapidMeta(app core.App, k vapidKeys) error {
	col, err := app.FindCollectionByNameOrId("app_meta")
	if err != nil {
		return err
	}
	rec, err := app.FindFirstRecordByFilter("app_meta", "key = {:k}", map[string]any{"k": vapidMetaKey})
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("key", vapidMetaKey)
	}
	rec.Set("value", map[string]any{"public": k.Public, "private": k.Private})
	return app.Save(rec)
}

// saveSubscription upserts a push subscription for the user, keyed by endpoint
// so re-subscribing on the same device updates rather than duplicates. A
// browser subscription is device-scoped, not account-scoped, so an existing
// endpoint is reassigned when another signed-in user registers it.
func saveSubscription(app core.App, userID, endpoint, p256dh, auth, ua string) error {
	col, err := app.FindCollectionByNameOrId(pushSubsCollection)
	if err != nil {
		return err
	}
	rec, err := app.FindFirstRecordByFilter(pushSubsCollection, "endpoint = {:e}", map[string]any{"e": endpoint})
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("endpoint", endpoint)
	}
	rec.Set("user", userID)
	rec.Set("p256dh", p256dh)
	rec.Set("auth", auth)
	rec.Set("userAgent", truncate(ua, 400))
	return app.Save(rec)
}

// deleteSubscriptionByEndpoint removes the caller's subscription on explicit
// unsubscribe. The user filter prevents one authenticated user from deleting
// another user's subscription by knowing its endpoint.
func deleteSubscriptionByEndpoint(app core.App, userID, endpoint string) {
	if rec, err := app.FindFirstRecordByFilter(pushSubsCollection,
		"endpoint = {:e} && user = {:u}",
		map[string]any{"e": endpoint, "u": userID}); err == nil {
		_ = app.Delete(rec)
	}
}

// userHasPushSubscription reports whether the user has at least one device
// registered for push.
func userHasPushSubscription(app core.App, userID string) bool {
	_, err := app.FindFirstRecordByFilter(pushSubsCollection, "user = {:u}", map[string]any{"u": userID})
	return err == nil
}

// sendPushToUser delivers a payload to every device the user has registered,
// pruning any subscription the push service reports as gone (404/410). Returns
// true if at least one delivery succeeded.
func sendPushToUser(app core.App, userID string, p pushPayload) bool {
	keys, err := vapid(app)
	if err != nil || keys.Public == "" || keys.Private == "" {
		log.Printf("[notifications] push not configured: %v", err)
		return false
	}
	subs, err := app.FindRecordsByFilter(pushSubsCollection, "user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil || len(subs) == 0 {
		return false
	}
	body, err := json.Marshal(p)
	if err != nil {
		return false
	}
	subject := vapidSubject(app)
	ok := false
	for _, s := range subs {
		sub := &webpush.Subscription{
			Endpoint: s.GetString("endpoint"),
			Keys:     webpush.Keys{P256dh: s.GetString("p256dh"), Auth: s.GetString("auth")},
		}
		resp, err := webpush.SendNotification(body, sub, &webpush.Options{
			Subscriber:      subject,
			VAPIDPublicKey:  keys.Public,
			VAPIDPrivateKey: keys.Private,
			TTL:             86400,
		})
		if err != nil {
			log.Printf("[notifications] push send error: %v", err)
			continue
		}
		status := resp.StatusCode
		_ = resp.Body.Close()
		switch {
		case status == http.StatusNotFound || status == http.StatusGone:
			_ = app.Delete(s) // subscription is dead; clean it up
		case status >= 200 && status < 300:
			ok = true
		default:
			log.Printf("[notifications] push endpoint returned %d", status)
		}
	}
	return ok
}
