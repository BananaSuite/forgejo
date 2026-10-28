// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"context"
	"net/http"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionsTasks(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ctx, resp := contexttest.MockContext(t, "admin/actions/tasks")
	ActionsTasks(ctx)

	assert.Equal(t, http.StatusOK, resp.Code)
	tasks := ctx.Data["Tasks"].([]*actions_model.ActionTask)
	assert.NotEmpty(t, tasks)
	// The fixture tasks point to a runner that is not in the fixtures, so
	// the map is empty here. The integration test checks the runner column.
	assert.IsType(t, map[int64]*actions_model.ActionRunner{}, ctx.Data["TaskRunners"])
	assert.True(t, ctx.Data["ShowTaskRunner"].(bool))
}

// Each step of ActionsTasks reads its own table. Hiding that table makes the
// step fail, and the page must answer with a server error.
func TestActionsTasksDatabaseErrors(t *testing.T) {
	for _, table := range []string{"action_task", "action_run_job", "action_runner"} {
		t.Run(table, func(t *testing.T) {
			require.NoError(t, unittest.PrepareTestDatabase())
			hideTable(t, table)

			ctx, resp := contexttest.MockContext(t, "admin/actions/tasks")
			ActionsTasks(ctx)

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
