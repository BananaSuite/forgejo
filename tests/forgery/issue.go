// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgery

import (
	"testing"

	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	issue_service "forgejo.org/services/issue"
	pull_service "forgejo.org/services/pull"

	"github.com/stretchr/testify/require"
)

func createIssue(t *testing.T, user *user_model.User, repo *repo_model.Repository, title, content string, isPull bool) *issues_model.Issue {
	t.Helper()
	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		Title:    title,
		Content:  content,
		PosterID: user.ID,
		Poster:   user,
		IsPull:   isPull,
	}

	err := issue_service.NewIssue(t.Context(), repo, issue, nil, nil, nil)
	require.NoError(t, err)

	return issue
}

func CreateIssue(t *testing.T, user *user_model.User, repo *repo_model.Repository, title, content string) *issues_model.Issue {
	return createIssue(t, user, repo, title, content, false)
}

func CreatePullRequest(t *testing.T, user *user_model.User, repo *repo_model.Repository, title, content string) *issues_model.Issue {
	return createIssue(t, user, repo, title, content, true)
}

// CreatePRWithFork creates a proper PR in DB, assuming default branches
func CreatePRWithFork(t *testing.T, forker *user_model.User, base, fork *repo_model.Repository) (*issues_model.PullRequest, *issues_model.Issue) {
	t.Helper()
	prIssue := &issues_model.Issue{
		RepoID:   base.ID,
		Title:    "PR",
		Content:  "Plz merge",
		PosterID: forker.ID,
		Poster:   forker,
		IsPull:   true,
	}

	pr := &issues_model.PullRequest{
		HeadRepoID: fork.ID,
		BaseRepoID: base.ID,
		HeadBranch: fork.DefaultBranch,
		BaseBranch: base.DefaultBranch,
		HeadRepo:   fork,
		BaseRepo:   base,
		Type:       issues_model.PullRequestGitea,
	}
	err := pull_service.NewPullRequest(t.Context(), base, prIssue, []int64{}, []string{}, pr, []int64{})
	require.NoError(t, err)
	return pr, prIssue
}

func CreateMilestone(t *testing.T, repo *repo_model.Repository, name, content string) *issues_model.Milestone {
	milestone := &issues_model.Milestone{
		RepoID:  repo.ID,
		Name:    name,
		Content: content,
	}
	err := issues_model.NewMilestone(t.Context(), milestone)
	require.NoError(t, err)

	return milestone
}
