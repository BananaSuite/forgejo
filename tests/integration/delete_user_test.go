// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/organization"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func assertUserDeleted(t *testing.T, userID int64, purged bool) {
	unittest.AssertNotExistsBean(t, &user_model.User{ID: userID})
	unittest.AssertNotExistsBean(t, &user_model.Follow{UserID: userID})
	unittest.AssertNotExistsBean(t, &user_model.Follow{FollowID: userID})
	unittest.AssertNotExistsBean(t, &repo_model.Repository{OwnerID: userID})
	unittest.AssertNotExistsBean(t, &access_model.Access{UserID: userID})
	unittest.AssertNotExistsBean(t, &organization.OrgUser{UID: userID})
	unittest.AssertNotExistsBean(t, &issues_model.IssueUser{UID: userID})
	unittest.AssertNotExistsBean(t, &organization.TeamUser{UID: userID})
	unittest.AssertNotExistsBean(t, &repo_model.Star{UID: userID})
	if purged {
		unittest.AssertNotExistsBean(t, &issues_model.Issue{PosterID: userID})
	}
}

func TestUserDeleteAccount(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user8")
	urlStr := fmt.Sprintf("/user/settings/account/delete?password=%s", userPassword)
	req := NewRequest(t, "POST", urlStr)
	session.MakeRequest(t, req, http.StatusSeeOther)

	assertUserDeleted(t, 8, false)
	unittest.CheckConsistencyFor(t, &user_model.User{})
}

// A deleted user must not stay logged in: the session that issued the deletion has to be ended.
func TestUserDeleteAccountEndsSession(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user8")

	// Sanity check: the session is authenticated before the deletion.
	session.MakeRequest(t, NewRequest(t, "GET", "/user/settings"), http.StatusOK)

	urlStr := fmt.Sprintf("/user/settings/account/delete?password=%s", userPassword)
	session.MakeRequest(t, NewRequest(t, "POST", urlStr), http.StatusSeeOther)
	assertUserDeleted(t, 8, false)

	// The same browser session must now be treated as anonymous.
	resp := session.MakeRequest(t, NewRequest(t, "GET", "/user/settings"), http.StatusSeeOther)
	assert.True(t, strings.HasSuffix(resp.Header().Get("Location"), "/user/login"),
		"expected redirect to login, got %q", resp.Header().Get("Location"))
}

func TestUserDeleteAccountStillOwnRepos(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	urlStr := fmt.Sprintf("/user/settings/account/delete?password=%s", userPassword)
	req := NewRequest(t, "POST", urlStr)
	session.MakeRequest(t, req, http.StatusSeeOther)

	// user should not have been deleted, because the user still owns repos
	unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
}
