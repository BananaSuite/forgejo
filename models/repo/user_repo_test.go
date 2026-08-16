// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoAssignees(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	users, err := repo_model.GetRepoAssignees(db.DefaultContext, repo2)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, int64(2), users[0].ID)

	repo21 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 21})
	users, err = repo_model.GetRepoAssignees(db.DefaultContext, repo21)
	require.NoError(t, err)
	if assert.Len(t, users, 3) {
		assert.ElementsMatch(t, []int64{15, 16, 18}, []int64{users[0].ID, users[1].ID, users[2].ID})
	}

	// do not return deactivated users
	require.NoError(t, user_model.UpdateUserCols(db.DefaultContext, &user_model.User{ID: 15, IsActive: false}, "is_active"))
	users, err = repo_model.GetRepoAssignees(db.DefaultContext, repo21)
	require.NoError(t, err)
	if assert.Len(t, users, 2) {
		assert.NotContains(t, []int64{users[0].ID, users[1].ID}, 15)
	}
}

func assertReviewers(t *testing.T, repoID, posterID int64, expectedUIDs []int64) {
	ctx := db.DefaultContext
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})

	reviewers, err := repo_model.GetReviewers(ctx, repo, posterID)
	require.NoError(t, err)

	assert.Len(t, reviewers, len(expectedUIDs))
	for i, u := range reviewers {
		assert.Equal(t, expectedUIDs[i], u.ID)
	}
}

func TestRepoGetReviewers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// public repository, including
	// - people with read access
	// - people watching pull requests
	// - members of the organization owning the repository
	// who are NOT the poster and active

	// repo 1 has 0 people with explicit access, 5 people watching PRs (including
	// user2 and one inactive user), owner user2
	assertReviewers(t, 1, 2, []int64{1, 11, 4})

	// poster which is a canidate reviewer
	assertReviewers(t, 1, 1, []int64{11, 4})

	// repo 23 has 2 people with explicit access, 0 people watching PRs, 4 people
	// being members of the org (including the two with access)
	assertReviewers(t, 23, 1, []int64{15, 18, 2, 29})

	// private repository, including
	// - people with read access
	// - the owner
	// who are NOT the poster and active

	// repo 6 has 0 people with explicit access, 0 people watching PRs, owned by
	// user 10
	assertReviewers(t, 6, 1, []int64{10})

	// repo 5 has 1 user with explicit access, 1 user watching PRs (without
	// access) 3 people with org access (including user 2), which will not be
	// counted
	assertReviewers(t, 5, 1, []int64{2})
}

func GetWatchedRepoIDsOwnedBy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 9})
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	repoIDs, err := repo_model.GetWatchedRepoIDsOwnedBy(db.DefaultContext, user1.ID, user2.ID)
	require.NoError(t, err)
	assert.Len(t, repoIDs, 1)
	assert.EqualValues(t, 1, repoIDs[0])
}

func TestGetUserStarCount(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	profileUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	t.Logf("profileUser: ID=%d Name=%s", profileUser.ID, profileUser.Name)
	count, err := repo_model.GetUserStarCount(t.Context(), profileUser, profileUser)
	t.Logf("owner view: count=%d err=%v", count, err)
	require.NoError(t, err)
	require.Equal(t, 5, count, "owner should see all their own stars")
	count, err = repo_model.GetUserStarCount(t.Context(), profileUser, nil)
	t.Logf("anonymous view: count=%d err=%v", count, err)
	require.NoError(t, err)
	require.Equal(t, 2, count, "anon should only see public repos with public owners")

	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	t.Logf("user5: ID=%d Name=%s", user5.ID, user5.Name)

	count, err = repo_model.GetUserStarCount(t.Context(), profileUser, user5)
	t.Logf("user5 view: count=%d err=%v", count, err)
	require.NoError(t, err)
	require.Equal(t, 2, count, "unrelated signed-in user should only see public repos")

	user15 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 15})
	t.Logf("user15: ID=%d Name=%s", user15.ID, user15.Name)

	count, err = repo_model.GetUserStarCount(t.Context(), profileUser, user15)
	t.Logf("user15 (collaborator) view: count=%d err=%v", count, err)
	require.NoError(t, err)
	require.Equal(t, 3, count, "collaborator should see public repos + their collab private repo")
	user20 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})
	t.Logf("user20: ID=%d Name=%s", user20.ID, user20.Name)

	count, err = repo_model.GetUserStarCount(t.Context(), profileUser, user20)
	t.Logf("user20 (org member) view: count=%d err=%v", count, err)
	require.NoError(t, err)
	require.Equal(t, 3, count, "org member should see public repos + org's private repo")
	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	t.Logf("user1 (admin): ID=%d Name=%s IsAdmin=%v", user1.ID, user1.Name, user1.IsAdmin)

	count, err = repo_model.GetUserStarCount(t.Context(), profileUser, user1)
	t.Logf("admin view: count=%d err=%v", count, err)
	require.NoError(t, err)
	require.Equal(t, 5, count,
		"site admin should see all starred repos, bypassing all privacy filters")
	user22 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 22})
	t.Logf("user22 (limited_org viewer): ID=%d Name=%s Visibility=%d",
		user22.ID, user22.Name, user22.Visibility)

	count, err = repo_model.GetUserStarCount(t.Context(), profileUser, user22)
	t.Logf("limited-org viewer: count=%d err=%v", count, err)
	require.NoError(t, err)
	require.Equal(t, 2, count,
		"a limited-visibility user with no special access sees only public repos, same as a stranger")
}
