package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// 0028 keeps deleted league-chat messages as placeholders for members while
// storing the original text in a hidden backend-only field for moderation and
// short-lived undo restores.
func init() {
	m.Register(func(app core.App) error {
		messages, err := app.FindCollectionByNameOrId(nLeagueMessages)
		if err != nil {
			return err
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		if f, ok := messages.Fields.GetByName("text").(*core.TextField); ok && f.Required {
			f.Required = false
		}
		if messages.Fields.GetByName("deleted") == nil {
			messages.Fields.Add(&core.BoolField{Name: "deleted"})
		}
		if messages.Fields.GetByName("deletedBy") == nil {
			messages.Fields.Add(&core.RelationField{Name: "deletedBy", CollectionId: users.Id, MaxSelect: 1})
		}
		if messages.Fields.GetByName("deletedAt") == nil {
			messages.Fields.Add(&core.DateField{Name: "deletedAt"})
		}
		if messages.Fields.GetByName("origText") == nil {
			messages.Fields.Add(&core.TextField{Name: "origText", Hidden: true, Max: 1000})
		}

		memberOfLeague := "@collection.league_members.league ?= league && @collection.league_members.user ?= @request.auth.id"
		messageVisible := "@request.auth.id != '' && league.inviteCode != 'GLOBAL' && " + memberOfLeague
		ownMessage := "user = @request.auth.id"
		serverOnlyDeleteFields := "@request.body.deleted:changed = false && @request.body.deletedBy:changed = false && @request.body.deletedAt:changed = false && @request.body.origText:changed = false"
		messages.UpdateRule = types.Pointer(messageVisible + " && " + ownMessage + " && deleted = false && @request.body.league:changed = false && @request.body.user:changed = false && " + serverOnlyDeleteFields)
		messages.DeleteRule = nil

		return app.Save(messages)
	}, func(app core.App) error {
		messages, err := app.FindCollectionByNameOrId(nLeagueMessages)
		if err != nil {
			return err
		}
		for _, name := range []string{"origText", "deletedAt", "deletedBy", "deleted"} {
			if messages.Fields.GetByName(name) != nil {
				messages.Fields.RemoveByName(name)
			}
		}
		if f, ok := messages.Fields.GetByName("text").(*core.TextField); ok && !f.Required {
			f.Required = true
		}

		memberOfLeague := "@collection.league_members.league ?= league && @collection.league_members.user ?= @request.auth.id"
		messageVisible := "@request.auth.id != '' && league.inviteCode != 'GLOBAL' && " + memberOfLeague
		ownMessage := "user = @request.auth.id"
		messages.UpdateRule = types.Pointer(messageVisible + " && " + ownMessage + " && @request.body.league:changed = false && @request.body.user:changed = false")
		messages.DeleteRule = types.Pointer(messageVisible + " && " + ownMessage)

		return app.Save(messages)
	})
}
