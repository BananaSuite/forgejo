// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"forgejo.org/models/db"
	org_model "forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/timeutil"
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
	publicTeam := unittest.AssertExistsAndLoadBean(t, &org_model.Team{ID: 1})
	publicOrg := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: publicTeam.OrgID})

	privateTeam := unittest.AssertExistsAndLoadBean(t, &org_model.Team{ID: 16})
	privateOrg := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: privateTeam.OrgID})

	// create an invitation for user10 to org3
	invite, err := org_model.CreateTeamInviteForUser(t.Context(), user2, user10, publicTeam)
	require.NoError(t, err)
	// create an invitation for user10 to the private org
	invitePrivateOrg, err := org_model.CreateTeamInviteForUser(t.Context(), user2, user10, privateTeam)
	require.NoError(t, err)
	// create an invitation for a deleted org/team (which will no longer be possible if/once we change those to foreign keys)
	inviteWithDeletedOrg := &org_model.TeamInvite{
		Token:       "abcdefg",
		InviterID:   user2.ID,
		OrgID:       123456,
		TeamID:      789012,
		Email:       "some_email@example.com",
		InvitedID:   optional.Some(user10.ID),
		InvitedUser: user10,
		ExpiryUnix:  optional.None[timeutil.TimeStamp](),
	}
	err = db.Insert(t.Context(), inviteWithDeletedOrg)
	require.NoError(t, err)

	session := loginUser(t, "user10")
	doc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/notifications"), http.StatusOK).Body)

	// the valid invitation to a public org is advertised on the notification page
	doc.AssertElement(t, fmt.Sprintf("a[href^='/org/invite/%s']", invite.Token), true)
	doc.AssertElement(t, fmt.Sprintf("a[href^='%s']", publicOrg.HomeLink()), true)
	// the valid invitation to a private org is advertised too, but without a link to the org
	doc.AssertElement(t, fmt.Sprintf("a[href^='/org/invite/%s']", invitePrivateOrg.Token), true)
	doc.AssertElement(t, fmt.Sprintf("a[href^='%s']", privateOrg.HomeLink()), false)
	// the invalid invitation is ignored
	doc.AssertElement(t, fmt.Sprintf("a[href^='/org/invite/%s']", inviteWithDeletedOrg.Token), false)
}
