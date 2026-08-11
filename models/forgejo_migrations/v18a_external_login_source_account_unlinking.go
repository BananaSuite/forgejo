// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "external login source account unlinking",
		Upgrade:     externalLoginSourceAccountUnlinking,
	})
}

func externalLoginSourceAccountUnlinking(x *xorm.Engine) error {
	// The original struct is named "Source" but is manually renamed to "login_source"
	type LoginSource struct {
		AllowUnlinking bool `xorm:"NOT NULL DEFAULT false"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(LoginSource))
	return err
}
