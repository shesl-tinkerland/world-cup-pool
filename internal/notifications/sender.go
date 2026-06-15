package notifications

import (
	"errors"
	"html"
	"net/mail"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

var (
	errSMTPDisabled = errors.New("smtp not enabled")
	errNoSender     = errors.New("sender address not configured")
	errUnknownEvent = errors.New("unknown notification event")
)

// rendered is a ready-to-send email.
type rendered struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

// renderData carries event-specific dynamic values used when building a
// notification. Fields not relevant to an event are ignored.
type renderData struct {
	// UntippedCount is the number of upcoming matches the user has not tipped,
	// used by EventUpcomingMatchesNotTipped.
	UntippedCount int
}

// pushPayload is the JSON body delivered to the service worker.
type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag"`
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// resolveLang normalises a language code to one we have copy for, defaulting to
// Norwegian Bokmål.
func resolveLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "en":
		return "en"
	case "nn":
		return "nn"
	default:
		return "nb"
	}
}

// appURL returns the base URL for links: PUBLIC_APP_URL env wins, otherwise the
// PocketBase application URL configured in settings.
func appURL(app core.App) string {
	if v := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_APP_URL")), "/"); v != "" {
		return v
	}
	return strings.TrimRight(app.Settings().Meta.AppURL, "/")
}

// render builds the email for an event in the given language.
func render(app core.App, event, lang string, data renderData) (rendered, error) {
	switch event {
	case EventPreKickoffReminder:
		return renderPreKickoff(resolveLang(lang), appURL(app)), nil
	case EventUpcomingMatchesNotTipped:
		return renderUpcoming(resolveLang(lang), appURL(app), data.UntippedCount), nil
	default:
		return rendered{}, errUnknownEvent
	}
}

// testPushPayload is the self-service "does push reach this device?" message,
// sent by POST /api/push/test. It bypasses prefs and the send log because the
// user explicitly asked for it.
func testPushPayload(app core.App, lang string) pushPayload {
	base := appURL(app)
	switch resolveLang(lang) {
	case "en":
		return pushPayload{Title: "Test notification", Body: "Push notifications are working on this device.", URL: base + "/settings", Tag: "push_test"}
	case "nn":
		return pushPayload{Title: "Testvarsel", Body: "Push-varsel fungerer på denne eininga.", URL: base + "/settings", Tag: "push_test"}
	default:
		return pushPayload{Title: "Testvarsel", Body: "Push-varsler fungerer på denne enheten.", URL: base + "/settings", Tag: "push_test"}
	}
}

// renderPushPayload builds the Web Push payload for an event, or false if the
// event has no push variant.
func renderPushPayload(app core.App, event, lang string, data renderData) (pushPayload, bool) {
	base := appURL(app)
	switch event {
	case EventPreKickoffReminder:
		l := resolveLang(lang)
		switch l {
		case "en":
			return pushPayload{Title: "The World Cup is about to kick off", Body: "Submit your tips before the first match.", URL: base + "/tips", Tag: event}, true
		case "nn":
			return pushPayload{Title: "VM startar snart", Body: "Lever tipsa dine før første kamp.", URL: base + "/tips", Tag: event}, true
		default:
			return pushPayload{Title: "VM starter snart", Body: "Lever tipsene dine før første kamp.", URL: base + "/tips", Tag: event}, true
		}
	case EventUpcomingMatchesNotTipped:
		l := resolveLang(lang)
		n := data.UntippedCount
		switch l {
		case "en":
			return pushPayload{Title: "Matches starting soon", Body: itoa(n) + " " + plural(n, "match", "matches") + " you haven't tipped kick off within a day.", URL: base + "/tips", Tag: event}, true
		case "nn":
			return pushPayload{Title: "Kampar startar snart", Body: itoa(n) + " " + plural(n, "kamp", "kampar") + " du ikkje har tipsa startar innan eitt døgn.", URL: base + "/tips", Tag: event}, true
		default:
			return pushPayload{Title: "Kamper starter snart", Body: itoa(n) + " " + plural(n, "kamp", "kamper") + " du ikke har tipset starter innen ett døgn.", URL: base + "/tips", Tag: event}, true
		}
	default:
		return pushPayload{}, false
	}
}

// sendEmail delivers one message via PocketBase SMTP. Returns a descriptive
// error if SMTP is not configured; callers decide whether that is fatal.
func sendEmail(app core.App, to, subject, htmlBody string) error {
	settings := app.Settings()
	if !settings.SMTP.Enabled {
		return errSMTPDisabled
	}
	senderAddr := settings.Meta.SenderAddress
	if senderAddr == "" {
		return errNoSender
	}
	msg := &mailer.Message{
		From:    mail.Address{Name: settings.Meta.SenderName, Address: senderAddr},
		To:      []mail.Address{{Address: to}},
		Subject: subject,
		HTML:    htmlBody,
	}
	return app.NewMailClient().Send(msg)
}

// layout wraps body content in an email-client-safe, World Cup-themed HTML
// shell: a deep-navy header with a gold "VM 26" crest and wordmark, a gold
// accent rule, the message body, an optional gold call-to-action button, and a
// footer. It is table-based with inline styles (no flex/grid, no SVG) so it
// renders consistently across mail clients, including Gmail.
func layout(lang, heading, intro, ctaLabel, ctaURL, footer string) string {
	kicker := "VM 2026"
	if lang == "en" {
		kicker = "World Cup 2026"
	}

	cta := ""
	if ctaLabel != "" && ctaURL != "" {
		cta = `<table role="presentation" cellpadding="0" cellspacing="0" style="margin:22px 0 4px;"><tr>` +
			`<td style="border-radius:10px;background:#ffcf3a;background:linear-gradient(180deg,#ffd95a,#f0b400);">` +
			`<a href="` + html.EscapeString(ctaURL) + `" style="display:inline-block;padding:13px 28px;font-size:16px;font-weight:700;color:#071019;text-decoration:none;border-radius:10px;">` +
			html.EscapeString(ctaLabel) + `</a></td></tr></table>`
	}

	return `<!DOCTYPE html>` +
		`<html><body style="margin:0;padding:0;background:#eceae5;font-family:'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#18181b;">` +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#eceae5;padding:24px 12px;"><tr><td align="center">` +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 8px 30px rgba(7,16,25,0.12);">` +
		// Header band (deep navy) with gold crest + wordmark.
		`<tr><td style="background:#071019;background:linear-gradient(135deg,#0e2330,#071019 72%);padding:30px 28px 22px;text-align:center;">` +
		`<table role="presentation" cellpadding="0" cellspacing="0" align="center"><tr>` +
		`<td width="58" height="58" align="center" valign="middle" style="background:linear-gradient(150deg,#fff7d6,#d9bb72 55%,#8e651e);border-radius:16px;color:#2b210a;font-weight:800;font-size:18px;line-height:1.05;">VM<br><span style="font-size:13px;">26</span></td>` +
		`</tr></table>` +
		`<div style="margin-top:14px;font-size:24px;font-weight:800;letter-spacing:0.3px;color:#f3f6ee;">VM&nbsp;Tipping&nbsp;🏆</div>` +
		`<div style="margin-top:5px;font-size:12px;color:#d9bb72;font-weight:600;letter-spacing:2px;text-transform:uppercase;">` + html.EscapeString(kicker) + `</div>` +
		`</td></tr>` +
		// Gold accent rule.
		`<tr><td style="height:4px;line-height:4px;font-size:0;background:linear-gradient(90deg,#ffe27a,#ffcf3a,#b88408);">&nbsp;</td></tr>` +
		// Body.
		`<tr><td style="padding:30px 30px 6px;">` +
		`<h1 style="margin:0 0 12px;font-size:20px;line-height:1.3;color:#071019;">` + html.EscapeString(heading) + `</h1>` +
		`<p style="margin:0;font-size:16px;line-height:1.55;color:#33373b;">` + html.EscapeString(intro) + `</p>` +
		cta +
		`</td></tr>` +
		// Footer.
		`<tr><td style="padding:6px 30px 30px;">` +
		`<hr style="border:none;border-top:1px solid #e9e7e2;margin:20px 0 14px;">` +
		`<p style="margin:0;color:#8a8a8a;font-size:12px;line-height:1.5;">` + html.EscapeString(footer) + `</p>` +
		`</td></tr></table>` +
		`<div style="max-width:520px;margin:14px auto 0;color:#9a9a93;font-size:11px;text-align:center;">VM Tipping</div>` +
		`</td></tr></table></body></html>`
}

// renderPreKickoff builds the pre-kickoff reminder in the given (resolved) language.
func renderPreKickoff(lang, base string) rendered {
	tipsURL := base + "/tips"
	switch lang {
	case "en":
		return rendered{
			Subject: "The World Cup is about to kick off — submit your tips!",
			HTML: layout(
				lang,
				"The World Cup is about to kick off",
				"Kickoff is near. Make sure you've submitted your tips so you don't miss out on points.",
				"Submit your tips", tipsURL,
				"You receive this because you turned on pre-kickoff reminders. You can turn them off in Settings.",
			),
		}
	case "nn":
		return rendered{
			Subject: "VM startar snart — hugs å levere tipsa dine!",
			HTML: layout(
				lang,
				"VM startar snart",
				"Det nærmar seg avspark. Sjå til at du har levert tipsa dine, så du ikkje går glipp av poeng.",
				"Lever tipsa dine", tipsURL,
				"Du får denne fordi du har slått på påminning før avspark. Du kan skru det av i Innstillingar.",
			),
		}
	default: // nb
		return rendered{
			Subject: "VM starter snart — husk å levere tipsene dine!",
			HTML: layout(
				lang,
				"VM starter snart",
				"Det nærmer seg avspark. Sørg for at du har levert tipsene dine, så du ikke går glipp av poeng.",
				"Lever tipsene dine", tipsURL,
				"Du får denne fordi du har slått på påminnelse før avspark. Du kan skru det av i Innstillinger.",
			),
		}
	}
}

// renderUpcoming builds the "matches starting soon you haven't tipped" reminder
// in the given (resolved) language. n is the number of untipped upcoming matches.
func renderUpcoming(lang, base string, n int) rendered {
	tipsURL := base + "/tips"
	switch lang {
	case "en":
		return rendered{
			Subject: "Matches starting soon — you haven't tipped them yet",
			HTML: layout(
				lang,
				itoa(n)+" "+plural(n, "match", "matches")+" still untipped",
				"You have "+itoa(n)+" "+plural(n, "match", "matches")+" kicking off within the next day that you haven't tipped. Submit them before kickoff so you don't miss out on points.",
				"Tip the matches", tipsURL,
				"You receive this because you turned on reminders for upcoming matches. You can turn them off in Settings.",
			),
		}
	case "nn":
		return rendered{
			Subject: "Kampar startar snart — du har ikkje tipsa enno",
			HTML: layout(
				lang,
				itoa(n)+" "+plural(n, "kamp", "kampar")+" utan tips",
				"Du har "+itoa(n)+" "+plural(n, "kamp", "kampar")+" som startar innan eitt døgn og som du ikkje har tipsa. Lever før avspark, så du ikkje går glipp av poeng.",
				"Tips kampane", tipsURL,
				"Du får denne fordi du har slått på påminning om komande kampar. Du kan skru det av i Innstillingar.",
			),
		}
	default: // nb
		return rendered{
			Subject: "Kamper starter snart — du har ikke tipset ennå",
			HTML: layout(
				lang,
				itoa(n)+" "+plural(n, "kamp", "kamper")+" uten tips",
				"Du har "+itoa(n)+" "+plural(n, "kamp", "kamper")+" som starter innen ett døgn og som du ikke har tipset. Lever før avspark, så du ikke går glipp av poeng.",
				"Tips kampene", tipsURL,
				"Du får denne fordi du har slått på påminnelse om kommende kamper. Du kan skru det av i Innstillinger.",
			),
		}
	}
}
