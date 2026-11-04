// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIListTeams(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Guest PoV with private teams list", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		MakeRequest(t, NewRequest(t, "GET", "/api/v1/orgs/org3/teams"), http.StatusUnauthorized)
	})

	t.Run("Guest PoV with public teams list", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/orgs/org6/teams"), http.StatusOK)

		teams := []*api.Team{}
		DecodeJSON(t, resp, &teams)
		// all teams are listed
		assert.Len(t, teams, 2)
	})
}
