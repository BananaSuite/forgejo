// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixtures have no jobs or tasks in this repository.
const filterTestRepoID = 9001

func insertFilterTestJob(t *testing.T, name string, status Status, runsOn ...string) *ActionRunJob {
	t.Helper()
	job := &ActionRunJob{RepoID: filterTestRepoID, OwnerID: 9002, Name: name, Status: status, RunsOn: runsOn, Handle: "filter-test-" + name}
	require.NoError(t, db.Insert(t.Context(), job))
	return job
}

func jobNames(jobs []*ActionRunJob) []string {
	names := make([]string, 0, len(jobs))
	for _, job := range jobs {
		names = append(names, job.Name)
	}
	return names
}

func TestFindActiveJobsOptions(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	insertFilterTestJob(t, "underscore", StatusWaiting, "a_b")
	insertFilterTestJob(t, "lookalike", StatusWaiting, "axb")
	insertFilterTestJob(t, "percent", StatusWaiting, "50%")
	insertFilterTestJob(t, "gpu", StatusRunning, "gpu", "linux")
	insertFilterTestJob(t, "done", StatusSuccess, "gpu")

	find := func(filter AdminJobFilter) []string {
		filter.RepoID = filterTestRepoID
		jobs, err := db.Find[ActionRunJob](t.Context(), FindActiveJobsOptions{AdminJobFilter: filter})
		require.NoError(t, err)
		return jobNames(jobs)
	}

	t.Run("No filter lists the unfinished jobs, newest first", func(t *testing.T) {
		assert.Equal(t, []string{"gpu", "percent", "lookalike", "underscore"}, find(AdminJobFilter{}))
	})
	t.Run("Label matches the whole label only", func(t *testing.T) {
		assert.Equal(t, []string{"underscore"}, find(AdminJobFilter{Label: "a_b"}))
		assert.Equal(t, []string{"percent"}, find(AdminJobFilter{Label: "50%"}))
		assert.Equal(t, []string{"gpu"}, find(AdminJobFilter{Label: "linux"}))
		assert.Empty(t, find(AdminJobFilter{Label: "gp"}))
		assert.Empty(t, find(AdminJobFilter{Label: "%"}))
	})
	t.Run("Status", func(t *testing.T) {
		assert.Equal(t, []string{"gpu"}, find(AdminJobFilter{Status: StatusRunning}))
		// A finished status is not one of the choices, so it is ignored.
		assert.Len(t, find(AdminJobFilter{Status: StatusSuccess}), 4)
	})
	t.Run("Created since", func(t *testing.T) {
		assert.Len(t, find(AdminJobFilter{Since: timeutil.TimeStampNow().Add(-60)}), 4)
		assert.Empty(t, find(AdminJobFilter{Since: timeutil.TimeStampNow().Add(3600)}))
	})
}

func TestFindFinishedTasksOptions(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	gpuJob := insertFilterTestJob(t, "gpu", StatusSuccess, "gpu")
	otherJob := insertFilterTestJob(t, "other", StatusFailure, "docker")
	addTask := func(job *ActionRunJob, status Status, stopped timeutil.TimeStamp) *ActionTask {
		task := &ActionTask{JobID: job.ID, RepoID: filterTestRepoID, Status: status, Stopped: stopped, TokenHash: fmt.Sprintf("filter-test-%d-%d", job.ID, stopped)}
		require.NoError(t, db.Insert(t.Context(), task))
		return task
	}
	now := timeutil.TimeStampNow()
	oldest := addTask(gpuJob, StatusSuccess, now-7200)
	newest := addTask(otherJob, StatusFailure, now-60)
	middle := addTask(gpuJob, StatusCancelled, now-3600)
	addTask(gpuJob, StatusRunning, 0)

	find := func(filter AdminJobFilter) []int64 {
		filter.RepoID = filterTestRepoID
		tasks, err := db.Find[ActionTask](t.Context(), FindFinishedTasksOptions{AdminJobFilter: filter})
		require.NoError(t, err)
		ids := make([]int64, 0, len(tasks))
		for _, task := range tasks {
			ids = append(ids, task.ID)
		}
		return ids
	}

	t.Run("No filter lists finished tasks, most recently finished first", func(t *testing.T) {
		assert.Equal(t, []int64{newest.ID, middle.ID, oldest.ID}, find(AdminJobFilter{}))
	})
	t.Run("Label of the job", func(t *testing.T) {
		assert.Equal(t, []int64{middle.ID, oldest.ID}, find(AdminJobFilter{Label: "gpu"}))
	})
	t.Run("Status", func(t *testing.T) {
		assert.Equal(t, []int64{middle.ID}, find(AdminJobFilter{Status: StatusCancelled}))
		// Running is not one of the choices, so it is ignored.
		assert.Len(t, find(AdminJobFilter{Status: StatusRunning}), 3)
	})
	t.Run("Done since", func(t *testing.T) {
		assert.Equal(t, []int64{newest.ID, middle.ID}, find(AdminJobFilter{Since: now - 3600}))
	})
}

func TestAdminFilterChoices(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	insertFilterTestJob(t, "waiting", StatusWaiting, "zz-label", "aa-label")
	insertFilterTestJob(t, "done", StatusSuccess, "done-only-label")

	labels, err := GetActiveJobLabels(ctx)
	require.NoError(t, err)
	assert.Contains(t, labels, "zz-label")
	assert.Contains(t, labels, "aa-label")
	assert.NotContains(t, labels, "done-only-label")
	assert.IsNonDecreasing(t, labels)

	repoIDs, err := GetActiveJobRepoIDs(ctx)
	require.NoError(t, err)
	assert.Contains(t, repoIDs, int64(filterTestRepoID))

	deleted := &ActionRunner{UUID: "c2f1a7e0-4d3b-4a8e-9f61-2b7d5e0c9001", Name: "deleted", TokenHash: "filter-test-deleted", AgentLabels: []string{"deleted-label"}}
	require.NoError(t, db.Insert(ctx, deleted))
	_, err = db.GetEngine(ctx).ID(deleted.ID).Delete(&ActionRunner{})
	require.NoError(t, err)

	runnerLabels, err := GetAllRunnerLabels(ctx)
	require.NoError(t, err)
	assert.Contains(t, runnerLabels, "deleted-label")
	assert.IsNonDecreasing(t, runnerLabels)

	require.NoError(t, db.Insert(ctx, &ActionTask{RepoID: filterTestRepoID, TokenHash: "filter-test-repo"}))
	taskRepoIDs, err := GetTaskRepoIDs(ctx)
	require.NoError(t, err)
	assert.Contains(t, taskRepoIDs, int64(filterTestRepoID))
}

func TestMissingRunnerLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	online := timeutil.TimeStamp(time.Now().Unix())
	offline := timeutil.TimeStamp(time.Now().Add(-2 * RunnerOfflineTime).Unix())
	count := 0
	addRunner := func(name string, repoID, ownerID int64, lastOnline timeutil.TimeStamp, labels ...string) {
		count++
		runner := &ActionRunner{UUID: fmt.Sprintf("d3e2b8f1-5e4c-4b9f-8a72-%012d", count), Name: name, RepoID: repoID, OwnerID: ownerID, LastOnline: lastOnline, AgentLabels: labels, TokenHash: "missing-" + name}
		require.NoError(t, db.Insert(ctx, runner))
	}
	addRunner("g", 0, 0, online, "linux")
	addRunner("rr", filterTestRepoID, 0, online, "repo-label")
	addRunner("own", 0, 9002, online, "owner-label")
	addRunner("offl", 0, 0, offline, "offline-label")
	addRunner("other", 12345, 0, online, "other-repo-label")

	jobs := []*ActionRunJob{
		insertFilterTestJob(t, "ok", StatusWaiting, "linux", "repo-label", "owner-label"),
		insertFilterTestJob(t, "offline", StatusWaiting, "linux", "offline-label"),
		insertFilterTestJob(t, "other-repo", StatusWaiting, "other-repo-label"),
		insertFilterTestJob(t, "expression", StatusWaiting, "${{ matrix.os }}"),
		insertFilterTestJob(t, "running", StatusRunning, "nowhere"),
	}

	missing, err := MissingRunnerLabels(ctx, jobs)
	require.NoError(t, err)
	assert.Equal(t, map[int64]string{
		jobs[1].ID: "offline-label",
		jobs[2].ID: "other-repo-label",
	}, missing)
}
