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

// activeJobsOptions finds the jobs that are not finished, newest first.
type activeJobsOptions struct {
	actions_model.FindRunJobOptions
}

func (opts activeJobsOptions) ToOrders() string {
	return "`id` DESC"
}

// ActionsJobs shows the jobs of all repositories that are waiting for a
// runner, blocked by another job, or running. A waiting job has no task yet,
// so the tasks page cannot show it.
func ActionsJobs(ctx *context.Context) {
	page := max(ctx.FormInt("page"), 1)

	opts := activeJobsOptions{actions_model.FindRunJobOptions{
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: jobsPageSize,
		},
		Statuses: []actions_model.Status{
			actions_model.StatusWaiting,
			actions_model.StatusBlocked,
			actions_model.StatusRunning,
		},
	}}

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

	ctx.Data["Title"] = ctx.Tr("actions.jobs")
	ctx.Data["PageType"] = "jobs"
	ctx.Data["PageIsSharedSettingsJobs"] = true
	ctx.Data["Jobs"] = jobs
	ctx.Data["JobRunners"] = runners
	ctx.Data["Total"] = count

	pager := context.NewPagination(int(count), opts.PageSize, opts.Page, 5)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplAdminActions)
}
