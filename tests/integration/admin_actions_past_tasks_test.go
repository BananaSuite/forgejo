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
	"forgejo.org/modules/timeutil"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminActionsPastTasks(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	ctx := t.Context()

	run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 791})
	job := &actions_model.ActionRunJob{RunID: run.ID, RepoID: run.RepoID, OwnerID: run.OwnerID, CommitSHA: run.CommitSHA, Name: "gpu-job", Status: actions_model.StatusSuccess, RunsOn: []string{"gpu", "linux"}, Handle: "admin-past-tasks-job"}
	require.NoError(t, db.Insert(ctx, job))

	liveRunner := &actions_model.ActionRunner{UUID: "c4d2a8f0-6e2b-4c1d-8b5e-9a7d3f2e1001", Name: "tasks-live-runner", TokenHash: "admin-tasks-live", AgentLabels: []string{"gpu", "linux"}}
	require.NoError(t, db.Insert(ctx, liveRunner))
	deletedRunner := &actions_model.ActionRunner{UUID: "c4d2a8f0-6e2b-4c1d-8b5e-9a7d3f2e1002", Name: "tasks-deleted-runner", TokenHash: "admin-tasks-deleted"}
	require.NoError(t, db.Insert(ctx, deletedRunner))
	_, err := db.GetEngine(ctx).ID(deletedRunner.ID).Delete(&actions_model.ActionRunner{})
	require.NoError(t, err)

	// 35 finished tasks of the gpu job, one minute apart, each 1m42s long.
	// The most recently finished one ran on the deleted runner.
	now := timeutil.TimeStampNow()
	const gpuTasks = 35
	for i := range gpuTasks {
		runnerID := liveRunner.ID
		if i == gpuTasks-1 {
			runnerID = deletedRunner.ID
		}
		stopped := now - timeutil.TimeStamp((gpuTasks-i)*60)
		task := &actions_model.ActionTask{
			JobID:     job.ID,
			RepoID:    run.RepoID,
			RunnerID:  runnerID,
			Status:    actions_model.StatusSuccess,
			Started:   stopped - 102,
			Stopped:   stopped,
			TokenHash: fmt.Sprintf("admin-tasks-%d", i),
		}
		require.NoError(t, db.Insert(ctx, task))
	}
	// A running task is not a past task.
	require.NoError(t, db.Insert(ctx, &actions_model.ActionTask{JobID: job.ID, RepoID: run.RepoID, RunnerID: liveRunner.ID, Status: actions_model.StatusRunning, Started: now, TokenHash: "admin-tasks-running"}))

	total, err := db.GetEngine(ctx).In("status", actions_model.FinishedTaskStatuses).Count(&actions_model.ActionTask{})
	require.NoError(t, err)

	t.Run("Admin sees the navbar entry and the finished tasks", func(t *testing.T) {
		session := loginUser(t, "user1")

		resp := session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/past-tasks"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		link := doc.Find(`.flex-container-nav a.active.item[href="/admin/actions/past-tasks"]`)
		assert.Equal(t, 1, link.Length())
		assert.Equal(t, "Past tasks", strings.TrimSpace(link.Text()))

		// The header shows the number of finished tasks on all pages.
		header := strings.TrimSpace(doc.Find(".task-list-container h4.header").Text())
		assert.Contains(t, header, fmt.Sprintf("(Total: %d)", total))

		rows := doc.Find(".task-list-container table tbody tr")
		assert.Equal(t, 30, rows.Length())
		assert.NotContains(t, doc.Find(".task-list-container table").Text(), "Running")

		// Most recently finished first: it ran on the deleted runner.
		first := rows.First().Find("td")
		assert.Equal(t, "gpu-job", strings.TrimSpace(first.Eq(2).Text()))
		assert.Equal(t, []string{"gpu", "linux"}, first.Eq(3).Find(".ui.label").Map(func(_ int, s *goquery.Selection) string { return strings.TrimSpace(s.Text()) }))
		assert.Equal(t, "tasks-deleted-runner (deleted)", strings.TrimSpace(first.Eq(4).Text()))
		assert.Equal(t, 0, first.Eq(4).Find("a").Length())
		assert.Equal(t, "1m42s", strings.TrimSpace(first.Eq(7).Text()))

		// The second task ran on the live runner and links to its page.
		runnerLink := rows.Eq(1).Find("td").Eq(4).Find("a")
		assert.Equal(t, "tasks-live-runner", strings.TrimSpace(runnerLink.Text()))
		href, _ := runnerLink.Attr("href")
		assert.Equal(t, fmt.Sprintf("/admin/actions/runners/%d", liveRunner.ID), href)

		assert.Positive(t, doc.Find(`.pagination a[href*="page=2"]`).Length())

		// The filters, with "Done at" as the time filter.
		filters := doc.Find(".task-list-container .filter.menu")
		assert.Equal(t, 4, filters.Find(".ui.dropdown").Length())
		assert.Equal(t, "Done at", strings.TrimSpace(filters.Find("#period_dropdown > .text").Text()))
		assert.Equal(t, 1, filters.Find(`#label_dropdown a[href*="label=gpu"]`).Length())
	})

	t.Run("Label filter counts and keeps the filter on the next page", func(t *testing.T) {
		session := loginUser(t, "user1")

		resp := session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/past-tasks?label=gpu"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		header := strings.TrimSpace(doc.Find(".task-list-container h4.header").Text())
		assert.Contains(t, header, fmt.Sprintf("(Total: %d)", gpuTasks))
		assert.Positive(t, doc.Find(`.pagination a[href*="label=gpu"][href*="page=2"]`).Length())

		resp = session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/past-tasks?label=gpu&page=2"), http.StatusOK)
		doc = NewHTMLParser(t, resp.Body)
		assert.Equal(t, gpuTasks-30, doc.Find(".task-list-container table tbody tr").Length())
	})

	t.Run("Done at filter", func(t *testing.T) {
		session := loginUser(t, "user1")

		// The fixture tasks finished in 2023; only the new ones are recent.
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/past-tasks?period=daily"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		header := strings.TrimSpace(doc.Find(".task-list-container h4.header").Text())
		assert.Contains(t, header, fmt.Sprintf("(Total: %d)", gpuTasks))
	})

	t.Run("Non-admin is refused", func(t *testing.T) {
		session := loginUser(t, "user2")
		session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/past-tasks"), http.StatusForbidden)
	})
}
