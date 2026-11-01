// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"net/http"
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/modules/test"
	"forgejo.org/modules/web"
	"forgejo.org/services/contexttest"
	"forgejo.org/services/forms"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetCommitNotesPullRequest(t *testing.T) {
	unittest.PrepareTestEnv(t)
	commitID := "65f1bf27bc3bf70f64657658635e66094edbcb4d"
	pullID := "5"
	path := "/user2/repo1/pulls/" + pullID + "/commits/" + commitID
	ctx, _ := contexttest.MockContext(t, path)
	ctx.SetParams(":sha", commitID)
	ctx.SetParams(":index", pullID)
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)
	notes := `This is a new note.\nSee https://frogejo.org.`
	web.SetForm(ctx, &forms.CommitNotesForm{
		Notes: notes,
	})
	SetCommitNotesPullRequest(ctx)
	assert.Equal(t, http.StatusSeeOther, ctx.Resp.Status())
	assert.Equal(t, path, test.RedirectURL(ctx.Resp))
	note, err := git.GetNote(ctx, ctx.Repo.GitRepo, commitID)
	require.NoError(t, err)
	assert.Equal(t, []byte(notes+"\n"), note.Message)
}

func TestRemoveCommitNotesPullRequest(t *testing.T) {
	unittest.PrepareTestEnv(t)
	commitID := "65f1bf27bc3bf70f64657658635e66094edbcb4d"
	pullID := "5"
	path := "/user2/repo1/pulls/" + pullID + "/commits/" + commitID
	ctx, _ := contexttest.MockContext(t, path)
	ctx.SetParams(":sha", commitID)
	ctx.SetParams(":index", pullID)
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)
	RemoveCommitNotesPullRequest(ctx)
	assert.Equal(t, http.StatusSeeOther, ctx.Resp.Status())
	assert.Equal(t, path, test.RedirectURL(ctx.Resp))
	note, err := git.GetNote(ctx, ctx.Repo.GitRepo, commitID)
	require.Error(t, err)
	assert.True(t, git.IsErrNotExist(err))
	assert.Nil(t, note)
}

// TestViewPullFilesMissingMergeBase is a regression test for a nil pointer
// dereference in viewPullFiles: when the compare info cannot be loaded for a
// merged pull request (here simulated by a merge base that does not resolve to
// a git object, the same failure mode a client disconnect can trigger when
// GetCompareInfo fails), PrepareMergedViewPullInfo reports the error via
// ctx.ServerError and returns a nil CompareInfo. The single commit view must
// bail out before dereferencing the nil CompareInfo while validating the
// requested commit, instead of panicking.
func TestViewPullFilesMissingMergeBase(t *testing.T) {
	unittest.PrepareTestEnv(t)

	const (
		pullID   = "2" // merged pull request in user2/repo1
		commitID = "65f1bf27bc3bf70f64657658635e66094edbcb4d"
		path     = "/user2/repo1/pulls/" + pullID + "/commits/" + commitID
	)

	// Point the merged pull request's merge base at a reference that does not
	// exist in the repository, so GetCompareInfo fails and
	// PrepareMergedViewPullInfo reports the error and returns nil.
	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{IssueID: 2})
	pull.MergeBase = "refs/heads/bogus-nonexistent-merge-base"
	_, err := db.GetEngine(db.DefaultContext).ID(pull.ID).Cols("merge_base").Update(pull)
	require.NoError(t, err)

	ctx, _ := contexttest.MockContext(t, path)
	ctx.SetParams(":sha", commitID)
	ctx.SetParams(":index", pullID)
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)

	assert.NotPanics(t, func() {
		ViewPullFilesForSingleCommit(ctx)
	})
	// PrepareMergedViewPullInfo already reported the CompareInfo failure as a
	// 500 ServerError; the handler must return quietly instead of panicking.
	assert.Equal(t, http.StatusInternalServerError, ctx.Resp.Status())
}
