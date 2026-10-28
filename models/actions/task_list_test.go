// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"context"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskList_LoadRunners(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	liveRunner := &ActionRunner{UUID: "f0c0e7a6-4a8e-4cf2-9b37-31a3b6c1a001", Name: "live-runner", TokenHash: "load-runners-live"}
	require.NoError(t, db.Insert(ctx, liveRunner))
	deletedRunner := &ActionRunner{UUID: "f0c0e7a6-4a8e-4cf2-9b37-31a3b6c1a002", Name: "deleted-runner", TokenHash: "load-runners-deleted"}
	require.NoError(t, db.Insert(ctx, deletedRunner))
	_, err := db.GetEngine(ctx).ID(deletedRunner.ID).Delete(&ActionRunner{})
	require.NoError(t, err)

	tasks := TaskList{
		{ID: 1, RunnerID: liveRunner.ID},
		{ID: 2, RunnerID: liveRunner.ID},
		{ID: 3, RunnerID: deletedRunner.ID},
		{ID: 4, RunnerID: 0},
		{ID: 5, RunnerID: 999999},
	}

	assert.ElementsMatch(t, []int64{liveRunner.ID, deletedRunner.ID, 999999}, tasks.GetRunnerIDs())

	runners, err := tasks.LoadRunners(ctx)
	require.NoError(t, err)
	require.Len(t, runners, 2)
	assert.Equal(t, "live-runner", runners[liveRunner.ID].Name)
	assert.Zero(t, runners[liveRunner.ID].Deleted)
	assert.Equal(t, "deleted-runner", runners[deletedRunner.ID].Name)
	assert.NotZero(t, runners[deletedRunner.ID].Deleted)
	assert.Nil(t, runners[999999])

	empty, err := TaskList{}.LoadRunners(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestTaskList_LoadRunnersDatabaseError(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// A cancelled context makes the database query fail.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	runners, err := TaskList{{ID: 1, RunnerID: 1}}.LoadRunners(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, runners)
}
