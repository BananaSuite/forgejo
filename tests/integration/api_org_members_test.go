// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIListMembers(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Guest PoV with private members list", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/orgs/org3/public_members"), http.StatusOK)

		members := []*api.User{}
		DecodeJSON(t, resp, &members)
		// only the members who made their membership public are listed
		assert.Len(t, members, 2)
	})

	t.Run("Guest PoV with public members list", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// hide the profile org badge of all members of org6
		require.NoError(t, organization.ChangeOrgUserStatus(db.DefaultContext, 6, 5, false))
		require.NoError(t, organization.ChangeOrgUserStatus(db.DefaultContext, 6, 28, false))

		resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/orgs/org6/public_members"), http.StatusOK)

		members := []*api.User{}
		DecodeJSON(t, resp, &members)
		// all members are listed
		assert.Len(t, members, 2)
	})
}
