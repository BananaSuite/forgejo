// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/services/context"
)

const jobsPageSize = 30

// ActionsActiveTasks shows the "Pending/active tasks" page: the jobs of all
// repositories that are waiting for a runner, blocked by another job, or
// running. A waiting job has no task yet, so the page lists jobs, not tasks.
func ActionsActiveTasks(ctx *context.Context) {
	page := max(ctx.FormInt("page"), 1)
	filter := readActionsFilter(ctx, actions_model.ActiveJobStatuses)

	opts := actions_model.FindActiveJobsOptions{
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: jobsPageSize,
		},
		AdminJobFilter: filter,
	}

	jobs, count, err := db.FindAndCount[actions_model.ActionRunJob](ctx, opts)
	if err != nil {
		ctx.ServerError("FindAndCount", err)
		return
	}

	jobList := actions_model.ActionJobList(jobs)
	if err := jobList.LoadAttributes(ctx, true); err != nil {
		ctx.ServerError("LoadAttributes", err)
		return
	}
	runners, err := jobList.LoadRunners(ctx)
	if err != nil {
		ctx.ServerError("LoadRunners", err)
		return
	}
	missing, err := actions_model.MissingRunnerLabels(ctx, jobs)
	if err != nil {
		ctx.ServerError("MissingRunnerLabels", err)
		return
	}

	labels, err := actions_model.GetActiveJobLabels(ctx)
	if err != nil {
		ctx.ServerError("GetActiveJobLabels", err)
		return
	}
	repoIDs, err := actions_model.GetActiveJobRepoIDs(ctx)
	if err != nil {
		ctx.ServerError("GetActiveJobRepoIDs", err)
		return
	}

	ctx.Data["Title"] = ctx.Tr("actions.active_tasks")
	ctx.Data["PageType"] = "active_tasks"
	ctx.Data["PageIsSharedSettingsActiveTasks"] = true
	ctx.Data["Jobs"] = jobs
	ctx.Data["JobRunners"] = runners
	ctx.Data["MissingLabels"] = missing
	ctx.Data["Total"] = count
	ctx.Data["PeriodTitle"] = ctx.Tr("actions.jobs.created")

	pager := context.NewPagination(int(count), opts.PageSize, opts.Page, 5)
	choices := actionsFilterChoices{statuses: actions_model.ActiveJobStatuses, labels: labels, repoIDs: repoIDs}
	if err := setActionsFilterChoices(ctx, choices, filter, pager); err != nil {
		ctx.ServerError("setActionsFilterChoices", err)
		return
	}
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplAdminActions)
}
