// Command wm-pickems is a single-binary WC 2026 prediction app: it runs the
// PocketBase backend (auth + SQLite + REST) and serves the embedded SvelteKit
// SPA from the same process, so the whole thing ships as one Docker image.
package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"

	"github.com/oyvhov/world-cup-pool/internal/account"
	"github.com/oyvhov/world-cup-pool/internal/chat"
	"github.com/oyvhov/world-cup-pool/internal/dev"
	"github.com/oyvhov/world-cup-pool/internal/forecast"
	"github.com/oyvhov/world-cup-pool/internal/leagues"
	"github.com/oyvhov/world-cup-pool/internal/notifications"
	"github.com/oyvhov/world-cup-pool/internal/oauth"
	wmOdds "github.com/oyvhov/world-cup-pool/internal/odds"
	"github.com/oyvhov/world-cup-pool/internal/scoring"
	"github.com/oyvhov/world-cup-pool/internal/seed"
	wmsync "github.com/oyvhov/world-cup-pool/internal/sync"
	"github.com/oyvhov/world-cup-pool/internal/tips"
	"github.com/oyvhov/world-cup-pool/internal/topscorer"
	"github.com/oyvhov/world-cup-pool/internal/web"
	_ "github.com/oyvhov/world-cup-pool/migrations"
)

func main() {
	app := pocketbase.New()

	// Go-code migrations (collections/schema live in ./migrations).
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo,
		Automigrate:  true,
	})

	// Seed teams/groups/fixtures from the embedded openfootball dataset on
	// first boot (idempotent).
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := seed.Run(e.App); err != nil {
			return err
		}

		if publicURL := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_APP_URL")), "/"); publicURL != "" {
			settings := e.App.Settings()
			if strings.TrimRight(settings.Meta.AppURL, "/") != publicURL {
				settings.Meta.AppURL = publicURL
				if err := e.App.Save(settings); err != nil {
					log.Printf("[boot] failed to sync Settings -> Application URL from PUBLIC_APP_URL: %v", err)
				} else {
					log.Printf("[boot] synced Settings -> Application URL from PUBLIC_APP_URL")
				}
			}
		}

		// If PB_ADMIN_EMAIL and PB_ADMIN_PASSWORD are provided in ENV, upsert the superuser
		if adminEmail := os.Getenv("PB_ADMIN_EMAIL"); adminEmail != "" {
			if adminPass := os.Getenv("PB_ADMIN_PASSWORD"); adminPass != "" {
				superusers, err := e.App.FindCollectionByNameOrId("_superusers")
				if err != nil {
					log.Printf("[boot] superusers collection not found: %v", err)
				} else {
					record, err := e.App.FindAuthRecordByEmail(superusers, adminEmail)
					if err != nil {
						record = core.NewRecord(superusers)
						record.SetEmail(adminEmail)
						record.SetPassword(adminPass)
						if err := e.App.Save(record); err != nil {
							log.Printf("[boot] failed to bootstrap superuser %s: %v", adminEmail, err)
						} else {
							log.Printf("Bootstrapped superuser: %s", adminEmail)
						}
					} else if !record.ValidatePassword(adminPass) {
						record.SetPassword(adminPass)
						if err := e.App.Save(record); err != nil {
							log.Printf("[boot] failed to update superuser password for %s: %v", adminEmail, err)
						} else {
							log.Printf("Updated superuser password for: %s", adminEmail)
						}
					}
				}
			}
		}

		account.Register(e.App, e)
		account.RegisterStats(e.App, e)
		account.RegisterPublicStats(e.App, e)
		oauth.Register(e.App)
		wmsync.Register(e.App, e)
		wmOdds.Register(e.App, e)
		topscorer.Register(e.App, e)
		leagues.Register(e.App, e)
		notifications.Register(e.App, e)
		notifications.StartCron(e.App)
		tips.Register(e.App, e)
		forecast.Register(e.App, e)
		scoring.Register(e.App, e)
		chat.Register(e.App, e)
		dev.Register(e.App, e)

		// Serve the web manifest with the correct MIME so it installs as a
		// proper PWA (apis.Static would send text/plain for .webmanifest).
		e.Router.GET("/manifest.webmanifest", func(re *core.RequestEvent) error {
			b, err := fs.ReadFile(web.DistFS(), "manifest.webmanifest")
			if err != nil {
				return apis.NewNotFoundError("", nil)
			}
			return re.Blob(200, "application/manifest+json", b)
		})
		web.RegisterInviteMetadata(e.App, e)
		return e.Next()
	})

	// Serve the embedded SvelteKit build with SPA (index.html) fallback so
	// client-side routes resolve. Registered last and only if no API/user
	// route already owns the path.
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				e.Router.GET("/{path...}", apis.Static(web.DistFS(), true))
			}
			return e.Next()
		},
		Priority: 999,
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
