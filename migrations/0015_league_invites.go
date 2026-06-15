package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const nLeagueInvites = "league_invites"

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		leagues, err := app.FindCollectionByNameOrId(nLeagues)
		if err != nil {
			return err
		}

		invites := core.NewBaseCollection(nLeagueInvites)
		invites.ListRule = nil
		invites.ViewRule = nil
		invites.CreateRule = nil
		invites.UpdateRule = nil
		invites.DeleteRule = nil
		invites.Fields.Add(&core.RelationField{Name: "league", CollectionId: leagues.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		invites.Fields.Add(&core.RelationField{Name: "invitedUser", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		invites.Fields.Add(&core.RelationField{Name: "invitedBy", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		invites.Fields.Add(&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"pending", "accepted", "declined"}})
		invites.Fields.Add(&core.DateField{Name: "actedAt"})
		invites.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		invites.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		invites.AddIndex("idx_li_pending_unique", true, "league, invitedUser", "status = 'pending'")
		invites.AddIndex("idx_li_invited_user_status", false, "invitedUser, status, created", "")
		invites.AddIndex("idx_li_league_status", false, "league, status, created", "")
		return app.Save(invites)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId(nLeagueInvites)
		if err == nil {
			return app.Delete(col)
		}
		return nil
	})
}
