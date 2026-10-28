// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"context"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/container"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionJobList_GetJobIDs(t *testing.T) {
	jobs := ActionJobList{
		&ActionRunJob{JobNamespace: "ns1", JobID: "job 1"},
		&ActionRunJob{JobNamespace: "ns1", JobID: "job 2"},
		&ActionRunJob{JobNamespace: "ns2", JobID: "job 1"},
		&ActionRunJob{JobNamespace: "ns2", JobID: "job 2"},
	}

	assert.Equal(t, container.SetOf(
		NamespacedJobIdentifier{Namespace: "ns1", Identifier: "job 2"},
		NamespacedJobIdentifier{Namespace: "ns1", Identifier: "job 1"},
		NamespacedJobIdentifier{Namespace: "ns2", Identifier: "job 2"},
		NamespacedJobIdentifier{Namespace: "ns2", Identifier: "job 1"},
	), jobs.GetJobIDs())
}

func TestActionJobList_LoadRunners(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	liveRunner := &ActionRunner{UUID: "6b1f0c52-9d3e-4f8a-a1c4-7e2d5b9f3001", Name: "live-runner", TokenHash: "job-runners-live"}
	require.NoError(t, db.Insert(ctx, liveRunner))
	deletedRunner := &ActionRunner{UUID: "6b1f0c52-9d3e-4f8a-a1c4-7e2d5b9f3002", Name: "deleted-runner", TokenHash: "job-runners-deleted"}
	require.NoError(t, db.Insert(ctx, deletedRunner))
	_, err := db.GetEngine(ctx).ID(deletedRunner.ID).Delete(&ActionRunner{})
	require.NoError(t, err)

	liveTask := &ActionTask{RunnerID: liveRunner.ID, TokenHash: "job-runners-task-live"}
	require.NoError(t, db.Insert(ctx, liveTask))
	deletedTask := &ActionTask{RunnerID: deletedRunner.ID, TokenHash: "job-runners-task-deleted"}
	require.NoError(t, db.Insert(ctx, deletedTask))
	goneRunnerTask := &ActionTask{RunnerID: 999999, TokenHash: "job-runners-task-gone"}
	require.NoError(t, db.Insert(ctx, goneRunnerTask))

	jobs := ActionJobList{
		{ID: 1, TaskID: liveTask.ID},
		{ID: 2, TaskID: deletedTask.ID},
		{ID: 3, TaskID: 0},                 // waiting: no runner took it yet
		{ID: 4, TaskID: goneRunnerTask.ID}, // its runner no longer exists
		{ID: 5, TaskID: 999999},            // its task no longer exists
	}

	assert.ElementsMatch(t, []int64{liveTask.ID, deletedTask.ID, goneRunnerTask.ID, 999999}, jobs.GetTaskIDs())

	runners, err := jobs.LoadRunners(ctx)
	require.NoError(t, err)
	require.Len(t, runners, 2)
	assert.Equal(t, "live-runner", runners[1].Name)
	assert.Zero(t, runners[1].Deleted)
	assert.Equal(t, "deleted-runner", runners[2].Name)
	assert.NotZero(t, runners[2].Deleted)

	empty, err := ActionJobList{{ID: 1}}.LoadRunners(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestActionJobList_LoadRunnersDatabaseError(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// A cancelled context makes the database query fail.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	runners, err := ActionJobList{{ID: 1, TaskID: 1}}.LoadRunners(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, runners)
}
