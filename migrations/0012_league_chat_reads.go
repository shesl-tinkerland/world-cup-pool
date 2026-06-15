package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

const nLeagueChatReads = "league_chat_reads"

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
		readVisible := "@request.auth.id != '' && user = @request.auth.id && league.inviteCode != 'GLOBAL' && " + memberOfLeague

		reads := core.NewBaseCollection(nLeagueChatReads)
		reads.ListRule = types.Pointer(readVisible)
		reads.ViewRule = types.Pointer(readVisible)
		reads.CreateRule = types.Pointer(readVisible)
		reads.UpdateRule = types.Pointer(readVisible + " && @request.body.league:changed = false && @request.body.user:changed = false")
		reads.DeleteRule = types.Pointer(readVisible)
		reads.Fields.Add(&core.RelationField{Name: "league", CollectionId: leagues.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		reads.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		reads.Fields.Add(&core.DateField{Name: "lastReadAt", Required: true})
		reads.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		reads.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		reads.AddIndex("idx_lcr_league_user", true, "league, user", "")
		return app.Save(reads)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId(nLeagueChatReads)
		if err == nil {
			return app.Delete(col)
		}
		return nil
	})
}
