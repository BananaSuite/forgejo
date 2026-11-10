// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"testing"

	org_model "forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/require"
)

func TestNotification(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)

	req := NewRequest(t, "GET", "/notifications")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	// Unread and pinned notification.
	htmlDoc.AssertElement(t, ".notifications-link[href='/user2/repo1/pulls/3']", true)
	htmlDoc.AssertElement(t, ".notifications-link[href='/user2/repo1/issues/4']", true)
	htmlDoc.AssertElement(t, ".notifications-link[href='/user2/repo2/issues/1']", true)

	// Read notification.
	htmlDoc.AssertElement(t, ".notifications-link[href='/user2/repo2/pulls/2']", false)
}

func TestTeamInviteNotification(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	user10 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10})
	ownersTeam := unittest.AssertExistsAndLoadBean(t, &org_model.Team{ID: 1})

	// create an invitation for user10 to org3
	invite, err := org_model.CreateTeamInviteForUser(t.Context(), user2, user10, ownersTeam)
	require.NoError(t, err)

	session := loginUser(t, "user10")
	doc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/notifications"), http.StatusOK).Body)

	// the invitation is advertised on the org page
	doc.AssertElement(t, fmt.Sprintf("a[href^='/org/invite/%s']", invite.Token), true)
}
