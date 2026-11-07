// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/modules/base"
	"forgejo.org/services/context"
)

const tplAdminActions base.TplName = "admin/actions"

const tasksPageSize = 30

// ActionsPastTasks shows the "Past tasks" page: the finished tasks of all
// runners on the instance, the most recently finished first.
func ActionsPastTasks(ctx *context.Context) {
	page := max(ctx.FormInt("page"), 1)
	filter := readActionsFilter(ctx, actions_model.FinishedTaskStatuses)

	opts := actions_model.FindFinishedTasksOptions{
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: tasksPageSize,
		},
		AdminJobFilter: filter,
	}

	tasks, count, err := db.FindAndCount[actions_model.ActionTask](ctx, opts)
	if err != nil {
		ctx.ServerError("FindAndCount", err)
		return
	}

	taskList := actions_model.TaskList(tasks)
	if err := taskList.LoadAttributes(ctx); err != nil {
		ctx.ServerError("LoadAttributes", err)
		return
	}
	runners, err := taskList.LoadRunners(ctx)
	if err != nil {
		ctx.ServerError("LoadRunners", err)
		return
	}

	labels, err := actions_model.GetAllRunnerLabels(ctx)
	if err != nil {
		ctx.ServerError("GetAllRunnerLabels", err)
		return
	}
	repoIDs, err := actions_model.GetTaskRepoIDs(ctx)
	if err != nil {
		ctx.ServerError("GetTaskRepoIDs", err)
		return
	}

	ctx.Data["Title"] = ctx.Tr("actions.past_tasks")
	ctx.Data["PageType"] = "past_tasks"
	ctx.Data["PageIsSharedSettingsPastTasks"] = true
	ctx.Data["Tasks"] = tasks
	ctx.Data["TaskRunners"] = runners
	ctx.Data["IsAdminTaskList"] = true
	ctx.Data["Total"] = count
	ctx.Data["PeriodTitle"] = ctx.Tr("actions.runners.task_list.done_at")

	pager := context.NewPagination(int(count), opts.PageSize, opts.Page, 5)
	choices := actionsFilterChoices{statuses: actions_model.FinishedTaskStatuses, labels: labels, repoIDs: repoIDs}
	if err := setActionsFilterChoices(ctx, choices, filter, pager); err != nil {
		ctx.ServerError("setActionsFilterChoices", err)
		return
	}
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplAdminActions)
}
