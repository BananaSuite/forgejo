// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add team visibility setting to organizations",
		Upgrade:     addTeamsVisibilitySetting,
	})
}

func addTeamsVisibilitySetting(x *xorm.Engine) error {
	type User struct {
		VisibleTeams bool `xorm:"NOT NULL DEFAULT false"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(User))
	return err
}
