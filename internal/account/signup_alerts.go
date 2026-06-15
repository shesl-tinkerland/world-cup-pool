package account

import (
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

const alertBotSuffix = "@dev.local"

// registerSignupAlerts wires an OnRecordAfterCreateSuccess hook for the users
// collection that sends a notification email to the configured admin whenever
// a new account is created (email/password and Google OAuth both trigger this).
//
// The recipient is taken from the SIGNUP_ALERT_EMAIL env var, falling back to
// PB_ADMIN_EMAIL. If neither is set, the hook is a no-op. Mail failures are
// logged but never surface as errors so a misconfigured SMTP cannot block signup.
func registerSignupAlerts(app core.App) {
	app.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		// Skip dev bots used by the local match simulation tooling.
		email := strings.ToLower(strings.TrimSpace(e.Record.GetString("email")))
		if strings.HasSuffix(email, alertBotSuffix) {
			return e.Next()
		}

		// Resolve recipient – evaluated at hook time so env changes take effect
		// after a restart without touching the binary.
		to := os.Getenv("SIGNUP_ALERT_EMAIL")
		if to == "" {
			to = os.Getenv("PB_ADMIN_EMAIL")
		}
		if to == "" {
			return e.Next()
		}

		// Snapshot before goroutine to avoid any data-race on the record.
		name := e.Record.GetString("name")
		id := e.Record.Id
		created := e.Record.GetString("created")

		// Run in background: slow or failed delivery must never block signup.
		go sendSignupAlert(app, to, id, name, email, created)

		return e.Next()
	})
}

// sendSignupAlert builds and sends the admin notification. All errors are
// logged only; nothing is returned or propagated.
func sendSignupAlert(app core.App, to, id, name, email, created string) {
	settings := app.Settings()

	if !settings.SMTP.Enabled {
		log.Printf("[signup-alert] SMTP not enabled – skipping alert for user %s", id)
		return
	}

	senderAddr := settings.Meta.SenderAddress
	if senderAddr == "" {
		log.Printf("[signup-alert] sender address not configured – skipping alert for user %s", id)
		return
	}

	displayName := name
	if displayName == "" {
		displayName = "(no name)"
	}

	subject := fmt.Sprintf("Ny brukar: %s (%s)", displayName, email)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;color:#222;max-width:480px;">
<h2 style="color:#1055c9;">Ny konto registrert</h2>
<table style="border-collapse:collapse;width:100%%;">
  <tr><td style="padding:4px 16px 4px 0;font-weight:bold;">Namn</td><td>%s</td></tr>
  <tr><td style="padding:4px 16px 4px 0;font-weight:bold;">E-post</td><td>%s</td></tr>
  <tr><td style="padding:4px 16px 4px 0;font-weight:bold;">Brukar-ID</td><td>%s</td></tr>
  <tr><td style="padding:4px 16px 4px 0;font-weight:bold;">Oppretta</td><td>%s</td></tr>
</table>
</body>
</html>`, displayName, email, id, created)

	msg := &mailer.Message{
		From: mail.Address{
			Name:    settings.Meta.SenderName,
			Address: senderAddr,
		},
		To:      []mail.Address{{Address: to}},
		Subject: subject,
		HTML:    html,
	}

	if err := app.NewMailClient().Send(msg); err != nil {
		log.Printf("[signup-alert] failed to deliver alert for user %s: %v", id, err)
		return
	}

	log.Printf("[signup-alert] alert sent for new user %s (%s) → %s", id, email, to)
}
