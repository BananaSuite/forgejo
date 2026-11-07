// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"context"
	"net/http"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionsPastTasks(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ctx, resp := contexttest.MockContext(t, "admin/actions/past-tasks")
	ActionsPastTasks(ctx)

	assert.Equal(t, http.StatusOK, resp.Code)
	tasks := ctx.Data["Tasks"].([]*actions_model.ActionTask)
	require.NotEmpty(t, tasks)
	for _, task := range tasks {
		assert.True(t, task.Status.IsDone(), "task %d has status %s", task.ID, task.Status)
	}
	assert.Equal(t, int64(len(tasks)), ctx.Data["Total"])
	// The fixture tasks point to a runner that is not in the fixtures, so
	// the map is empty here. The integration test checks the runner column.
	assert.IsType(t, map[int64]*actions_model.ActionRunner{}, ctx.Data["TaskRunners"])
	assert.True(t, ctx.Data["IsAdminTaskList"].(bool))
	assert.True(t, ctx.Data["PageIsSharedSettingsPastTasks"].(bool))
	assert.Len(t, ctx.Data["StatusOptions"], len(actions_model.FinishedTaskStatuses))
	assert.NotEmpty(t, ctx.Data["RepoOptions"].([]*repo_model.Repository))
	assert.Equal(t, actionsPeriods, ctx.Data["PeriodOptions"])
}

func TestActionsPastTasksFilters(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ctx, resp := contexttest.MockContext(t, "admin/actions/past-tasks?status=1&label=x_y&repo=4&period=weekly&page=1")
	ActionsPastTasks(ctx)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, 1, ctx.Data["CurStatus"])
	assert.Equal(t, "x_y", ctx.Data["CurLabel"])
	assert.Equal(t, int64(4), ctx.Data["CurRepo"])
	assert.Equal(t, "weekly", ctx.Data["CurPeriod"])
	// The fixture tasks finished in 2023, so none is in the last week.
	assert.Empty(t, ctx.Data["Tasks"])
}

func TestActionsFilterIgnoresUnknownValues(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Running (6) is not a status of a past task, and "forever" is not a period.
	ctx, resp := contexttest.MockContext(t, "admin/actions/past-tasks?status=6&period=forever")
	ActionsPastTasks(ctx)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, 0, ctx.Data["CurStatus"])
	assert.Empty(t, ctx.Data["CurPeriod"])
	assert.NotEmpty(t, ctx.Data["Tasks"])
}

// Each step of the page reads its own table. Hiding that table makes the
// step fail, and the page must answer with a server error.
func TestActionsPastTasksDatabaseErrors(t *testing.T) {
	for _, table := range []string{"action_task", "action_run_job", "action_runner", "repository"} {
		t.Run(table, func(t *testing.T) {
			require.NoError(t, unittest.PrepareTestDatabase())
			hideTable(t, table)

			ctx, resp := contexttest.MockContext(t, "admin/actions/past-tasks")
			ActionsPastTasks(ctx)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	}
}

// hideTable renames a table for the duration of the test.
func hideTable(t *testing.T, table string) {
	t.Helper()
	hidden := table + "_hidden"
	_, err := db.GetEngine(t.Context()).Exec("ALTER TABLE " + table + " RENAME TO " + hidden)
	require.NoError(t, err)
	t.Cleanup(func() {
		// t.Context() is already cancelled when cleanup runs.
		_, err := db.GetEngine(context.Background()).Exec("ALTER TABLE " + hidden + " RENAME TO " + table)
		require.NoError(t, err)
	})
}

func TestActionsActiveTasks(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ctx, resp := contexttest.MockContext(t, "admin/actions/pending-active-tasks")
	ActionsActiveTasks(ctx)

	assert.Equal(t, http.StatusOK, resp.Code)
	jobs := ctx.Data["Jobs"].([]*actions_model.ActionRunJob)
	require.NotEmpty(t, jobs)
	for _, job := range jobs {
		assert.Contains(t, actions_model.ActiveJobStatuses, job.Status)
		assert.NotNil(t, job.Run)
	}
	assert.Equal(t, int64(len(jobs)), ctx.Data["Total"])
	assert.IsType(t, map[int64]*actions_model.ActionRunner{}, ctx.Data["JobRunners"])
	assert.True(t, ctx.Data["PageIsSharedSettingsActiveTasks"].(bool))
	assert.Len(t, ctx.Data["StatusOptions"], len(actions_model.ActiveJobStatuses))
	assert.NotEmpty(t, ctx.Data["LabelOptions"])

	// The fixtures have no online runner, so every waiting job with labels
	// gets a warning.
	missing := ctx.Data["MissingLabels"].(map[int64]string)
	for _, job := range jobs {
		if job.Status == actions_model.StatusWaiting && len(job.RunsOn) > 0 {
			assert.Equal(t, job.RunsOn[0], missing[job.ID])
		}
	}
}

func TestActionsActiveTasksFilters(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ctx, resp := contexttest.MockContext(t, "admin/actions/pending-active-tasks?label=fedora")
	ActionsActiveTasks(ctx)

	assert.Equal(t, http.StatusOK, resp.Code)
	jobs := ctx.Data["Jobs"].([]*actions_model.ActionRunJob)
	require.NotEmpty(t, jobs)
	for _, job := range jobs {
		assert.Contains(t, job.RunsOn, "fedora")
	}
}

func TestActionsActiveTasksDatabaseErrors(t *testing.T) {
	for _, table := range []string{"action_run_job", "action_run", "action_task", "action_runner", "repository"} {
		t.Run(table, func(t *testing.T) {
			require.NoError(t, unittest.PrepareTestDatabase())
			if table == "action_task" {
				// Only a job with a task makes the page read the task table.
				task := &actions_model.ActionTask{TokenHash: "admin-jobs-task"}
				require.NoError(t, db.Insert(t.Context(), task))
				_, err := db.GetEngine(t.Context()).Exec("UPDATE action_run_job SET status = ?, task_id = ? WHERE id = (SELECT MIN(id) FROM action_run_job)", actions_model.StatusRunning, task.ID)
				require.NoError(t, err)
			}
			hideTable(t, table)

			ctx, resp := contexttest.MockContext(t, "admin/actions/pending-active-tasks")
			ActionsActiveTasks(ctx)

			assert.Equal(t, http.StatusInternalServerError, resp.Code)
		})
	}
}
