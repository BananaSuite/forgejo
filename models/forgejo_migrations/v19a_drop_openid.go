// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "drop OpenID authentication support",
		Upgrade:     dropTableUserOpenID,
	})
}

func dropTableUserOpenID(x *xorm.Engine) error {
	type UserOpenID struct {
		ID   int64  `xorm:"pk autoincr"`
		UID  int64  `xorm:"INDEX NOT NULL"`
		URI  string `xorm:"UNIQUE NOT NULL"`
		Show bool   `xorm:"DEFAULT false"`
	}
	return x.DropTables(new(UserOpenID))
}
