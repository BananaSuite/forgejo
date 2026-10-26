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

// ActionsTasks shows the tasks of all runners on the instance, newest first:
// waiting, running and finished ones.
func ActionsTasks(ctx *context.Context) {
	page := max(ctx.FormInt("page"), 1)

	opts := actions_model.FindTaskOptions{
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: tasksPageSize,
		},
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

	ctx.Data["Title"] = ctx.Tr("actions.tasks")
	ctx.Data["PageType"] = "tasks"
	ctx.Data["PageIsSharedSettingsTasks"] = true
	ctx.Data["Tasks"] = tasks
	ctx.Data["TaskRunners"] = runners
	ctx.Data["ShowTaskRunner"] = true

	pager := context.NewPagination(int(count), opts.PageSize, opts.Page, 5)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplAdminActions)
}
