// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add members and team visibility settings to organizations",
		Upgrade:     addMembersTeamsVisibilitySettings,
	})
}

func addMembersTeamsVisibilitySettings(x *xorm.Engine) error {
	type User struct {
		VisibleMembers bool `xorm:"NOT NULL DEFAULT false"`
		VisibleTeams   bool `xorm:"NOT NULL DEFAULT false"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(User))
	return err
}
