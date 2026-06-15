package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	nLeagueMessages         = "league_messages"
	nLeagueMessageReactions = "league_message_reactions"
)

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

		memberOfLeague := "@collection.league_members.league ?= league && @collection.league_members.user ?= @request.auth.id"
		messageVisible := "@request.auth.id != '' && league.inviteCode != 'GLOBAL' && " + memberOfLeague
		ownMessage := "user = @request.auth.id"

		messages := core.NewBaseCollection(nLeagueMessages)
		messages.ListRule = types.Pointer(messageVisible)
		messages.ViewRule = types.Pointer(messageVisible)
		messages.CreateRule = types.Pointer(messageVisible + " && " + ownMessage)
		messages.UpdateRule = types.Pointer(messageVisible + " && " + ownMessage + " && @request.body.league:changed = false && @request.body.user:changed = false")
		messages.DeleteRule = types.Pointer(messageVisible + " && " + ownMessage)
		messages.Fields.Add(&core.RelationField{Name: "league", CollectionId: leagues.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		messages.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		messages.Fields.Add(&core.TextField{Name: "text", Required: true, Max: 1000})
		messages.Fields.Add(&core.DateField{Name: "editedAt"})
		messages.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		messages.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		messages.AddIndex("idx_league_messages_league_created", false, "league, created", "")
		if err := app.Save(messages); err != nil {
			return err
		}

		memberOfMessageLeague := "@collection.league_members.league ?= message.league && @collection.league_members.user ?= @request.auth.id"
		reactionVisible := "@request.auth.id != '' && message.league.inviteCode != 'GLOBAL' && " + memberOfMessageLeague
		ownReaction := "user = @request.auth.id"

		reactions := core.NewBaseCollection(nLeagueMessageReactions)
		reactions.ListRule = types.Pointer(reactionVisible)
		reactions.ViewRule = types.Pointer(reactionVisible)
		reactions.CreateRule = types.Pointer(reactionVisible + " && " + ownReaction)
		reactions.UpdateRule = nil
		reactions.DeleteRule = types.Pointer(reactionVisible + " && " + ownReaction)
		reactions.Fields.Add(&core.RelationField{Name: "message", CollectionId: messages.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		reactions.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		reactions.Fields.Add(&core.TextField{Name: "emoji", Required: true, Max: 16})
		reactions.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		reactions.AddIndex("idx_lmr_message_user_emoji", true, "message, user, emoji", "")
		reactions.AddIndex("idx_lmr_message_created", false, "message, created", "")
		return app.Save(reactions)
	}, func(app core.App) error {
		for _, name := range []string{nLeagueMessageReactions, nLeagueMessages} {
			col, err := app.FindCollectionByNameOrId(name)
			if err == nil {
				if err := app.Delete(col); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
