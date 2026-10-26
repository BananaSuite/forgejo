// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminActionsTasks(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	ctx := t.Context()

	// The fixtures hold 12 tasks. Add tasks on two new runners, one of
	// them deleted, so that the list needs a second page (30 per page).
	liveRunner := &actions_model.ActionRunner{UUID: "c4d2a8f0-6e2b-4c1d-8b5e-9a7d3f2e1001", Name: "tasks-live-runner", TokenHash: "admin-tasks-live"}
	require.NoError(t, db.Insert(ctx, liveRunner))
	deletedRunner := &actions_model.ActionRunner{UUID: "c4d2a8f0-6e2b-4c1d-8b5e-9a7d3f2e1002", Name: "tasks-deleted-runner", TokenHash: "admin-tasks-deleted"}
	require.NoError(t, db.Insert(ctx, deletedRunner))
	_, err := db.GetEngine(ctx).ID(deletedRunner.ID).Delete(&actions_model.ActionRunner{})
	require.NoError(t, err)

	for i := range 25 {
		runnerID := liveRunner.ID
		if i == 24 {
			runnerID = deletedRunner.ID
		}
		task := &actions_model.ActionTask{
			RunnerID:  runnerID,
			Status:    actions_model.StatusRunning,
			TokenHash: fmt.Sprintf("admin-tasks-%d", i),
		}
		require.NoError(t, db.Insert(ctx, task))
	}
	total := unittest.GetCount(t, &actions_model.ActionTask{})
	require.Equal(t, 37, total)

	t.Run("Admin sees the navbar entry and all tasks", func(t *testing.T) {
		session := loginUser(t, "user1")

		resp := session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/tasks"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		link := doc.Find(`.flex-container-nav a.active.item[href="/admin/actions/tasks"]`)
		assert.Equal(t, 1, link.Length())

		rows := doc.Find(".task-list-container table tbody tr")
		assert.Equal(t, 30, rows.Length())

		// Newest task first: it ran on the deleted runner.
		firstRunner := strings.TrimSpace(rows.First().Find("td").Eq(2).Text())
		assert.Equal(t, "tasks-deleted-runner (deleted)", firstRunner)
		assert.Equal(t, 0, rows.First().Find("td").Eq(2).Find("a").Length())

		// The second task ran on the live runner and links to its page.
		runnerLink := rows.Eq(1).Find("td").Eq(2).Find("a")
		assert.Equal(t, "tasks-live-runner", strings.TrimSpace(runnerLink.Text()))
		href, _ := runnerLink.Attr("href")
		assert.Equal(t, fmt.Sprintf("/admin/actions/runners/%d", liveRunner.ID), href)

		assert.Positive(t, doc.Find(`.pagination a[href*="page=2"]`).Length())
	})

	t.Run("Second page shows the remaining tasks", func(t *testing.T) {
		session := loginUser(t, "user1")

		resp := session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/tasks?page=2"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		rows := doc.Find(".task-list-container table tbody tr")
		assert.Equal(t, total-30, rows.Length())

		// Fixture tasks ran on a runner that no longer exists at all.
		assert.Equal(t, "—", strings.TrimSpace(rows.First().Find("td").Eq(2).Text()))
	})

	t.Run("Non-admin is refused", func(t *testing.T) {
		session := loginUser(t, "user2")
		session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/tasks"), http.StatusForbidden)
	})
}
