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

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminActionsPendingActiveTasks(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	ctx := t.Context()

	run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 791})
	require.NoError(t, run.LoadRepo(ctx))

	liveRunner := &actions_model.ActionRunner{UUID: "5e8b2d14-7c3a-4f9e-b6d1-0a4c8e2f7001", Name: "jobs-live-runner", TokenHash: "admin-jobs-live"}
	require.NoError(t, db.Insert(ctx, liveRunner))
	deletedRunner := &actions_model.ActionRunner{UUID: "5e8b2d14-7c3a-4f9e-b6d1-0a4c8e2f7002", Name: "jobs-deleted-runner", TokenHash: "admin-jobs-deleted"}
	require.NoError(t, db.Insert(ctx, deletedRunner))
	_, err := db.GetEngine(ctx).ID(deletedRunner.ID).Delete(&actions_model.ActionRunner{})
	require.NoError(t, err)

	// Jobs are listed newest first, so insert them in reverse order of the
	// rows checked below. The finished job must not be listed at all.
	addJob := func(name string, status actions_model.Status, runnerID int64, runsOn ...string) {
		job := &actions_model.ActionRunJob{RunID: run.ID, RepoID: run.RepoID, OwnerID: run.OwnerID, CommitSHA: run.CommitSHA, Name: name, Status: status, RunsOn: runsOn, Handle: "admin-jobs-" + name}
		if runnerID != 0 {
			task := &actions_model.ActionTask{RunnerID: runnerID, Status: status, TokenHash: "admin-jobs-" + name}
			require.NoError(t, db.Insert(ctx, task))
			job.TaskID = task.ID
		}
		require.NoError(t, db.Insert(ctx, job))
	}
	addJob("job-waiting", actions_model.StatusWaiting, 0, "admin-test-gpu", "admin-test-linux")
	addJob("job-on-deleted-runner", actions_model.StatusRunning, deletedRunner.ID)
	addJob("job-on-live-runner", actions_model.StatusRunning, liveRunner.ID, "admin-test-linux")
	addJob("job-finished", actions_model.StatusSuccess, liveRunner.ID)

	active, err := db.GetEngine(ctx).In("status", actions_model.ActiveJobStatuses).Count(&actions_model.ActionRunJob{})
	require.NoError(t, err)

	t.Run("Admin sees the navbar entry and the unfinished jobs", func(t *testing.T) {
		session := loginUser(t, "user1")

		resp := session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/pending-active-tasks"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		link := doc.Find(`.flex-container-nav a.active.item[href="/admin/actions/pending-active-tasks"]`)
		assert.Equal(t, 1, link.Length())
		assert.Equal(t, "Pending/active tasks", strings.TrimSpace(link.Text()))

		header := strings.TrimSpace(doc.Find(".job-list-container h4.header").Text())
		assert.Contains(t, header, fmt.Sprintf("(Total: %d)", active))

		rows := doc.Find(".job-list-container table tbody tr")
		assert.EqualValues(t, min(active, 30), rows.Length())
		assert.NotContains(t, doc.Find(".job-list-container table").Text(), "job-finished")

		first := rows.Eq(0).Find("td")
		assert.Equal(t, "job-on-live-runner", strings.TrimSpace(first.Eq(0).Text()))
		jobHref, _ := first.Eq(0).Find("a").Attr("href")
		assert.Equal(t, run.Link(), jobHref)
		runnerHref, _ := first.Eq(3).Find("a").Attr("href")
		assert.Equal(t, fmt.Sprintf("/admin/actions/runners/%d", liveRunner.ID), runnerHref)

		second := rows.Eq(1).Find("td")
		assert.Equal(t, "jobs-deleted-runner (deleted)", strings.TrimSpace(second.Eq(3).Text()))

		third := rows.Eq(2).Find("td")
		assert.Equal(t, "job-waiting", strings.TrimSpace(third.Eq(0).Text()))
		assert.Equal(t, "Waiting", strings.TrimSpace(third.Eq(1).Text()))
		assert.Equal(t, "—", strings.TrimSpace(third.Eq(3).Text()))

		// Labels, and a warning: no online runner has the first label.
		labels := third.Eq(2).Find(".ui.label").Map(func(_ int, s *goquery.Selection) string { return strings.TrimSpace(s.Text()) })
		assert.Equal(t, []string{"admin-test-gpu", "admin-test-linux"}, labels)
		warning, _ := third.Eq(2).Find("[data-tooltip-content]").Attr("data-tooltip-content")
		assert.Equal(t, "No matching online runner with label: admin-test-gpu", warning)
		// A running job gets no warning.
		assert.Equal(t, 0, first.Eq(2).Find("[data-tooltip-content]").Length())

		filters := doc.Find(".job-list-container .filter.menu")
		assert.Equal(t, 4, filters.Find(".ui.dropdown").Length())
		assert.Equal(t, "Created", strings.TrimSpace(filters.Find("#period_dropdown > .text").Text()))
		assert.Equal(t, 1, filters.Find(`#label_dropdown a[href*="label=admin-test-gpu"]`).Length())
	})

	t.Run("Filters", func(t *testing.T) {
		session := loginUser(t, "user1")

		resp := session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/pending-active-tasks?label=admin-test-linux"), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		header := strings.TrimSpace(doc.Find(".job-list-container h4.header").Text())
		assert.Contains(t, header, "(Total: 2)")

		resp = session.MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/admin/actions/pending-active-tasks?label=admin-test-linux&status=%d", actions_model.StatusWaiting)), http.StatusOK)
		doc = NewHTMLParser(t, resp.Body)
		rows := doc.Find(".job-list-container table tbody tr")
		require.Equal(t, 1, rows.Length())
		assert.Equal(t, "job-waiting", strings.TrimSpace(rows.First().Find("td").First().Text()))
	})

	t.Run("Non-admin is refused", func(t *testing.T) {
		session := loginUser(t, "user2")
		session.MakeRequest(t, NewRequest(t, "GET", "/admin/actions/pending-active-tasks"), http.StatusForbidden)
	})
}
