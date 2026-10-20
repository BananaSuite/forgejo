// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package permissions

import (
	"net/http"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
)

// GetUserRepoPermission returns the permission of the request's doer on the
// given repository. In contrast to access_model.GetUserRepoPermission it
// takes the authentication method into account: requests authenticated with
// an actions task token are not acting with the collaborator-based
// permissions of the synthetic forgejo-actions user. They derive permission
// from the task the token belongs to instead: write access to the repository
// the task runs in, or read access if it is a fork pull request.
func GetUserRepoPermission(ctx Context, repo *repo_model.Repository) (access_model.Permission, error) {
	if ctx.Doer() != nil && ctx.Doer().ID == user_model.ActionsUserID && ctx.Authentication().ActionsTaskID().Has() {
		_, taskID := ctx.Authentication().ActionsTaskID().Get()
		task, err := actions_model.GetTaskByID(ctx.Context(), taskID)
		if err != nil {
			return access_model.Permission{}, err
		}
		return access_model.GetActionRepoPermission(ctx.Context(), repo, task)
	}
	return access_model.GetUserRepoPermissionWithReducer(ctx.Context(), repo, ctx.Doer(), ctx.Reducer())
}

func RepoAccess(ctx Context) {
	if ctx.Doer() != nil && ctx.Doer().ID == user_model.ActionsUserID && ctx.Authentication().ActionsTaskID().Has() {
		_, taskID := ctx.Authentication().ActionsTaskID().Get()
		task, err := actions_model.GetTaskByID(ctx.Context(), taskID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "actions_model.GetTaskByID", err)
			return
		}
		if task.RepoID != ctx.Repository().ID {
			ctx.NotFound()
			return
		}

		if task.IsForkPullRequest {
			ctx.Permission().AccessMode = perm.AccessModeRead
		} else {
			ctx.Permission().AccessMode = perm.AccessModeWrite
		}

		if err := ctx.Repository().LoadUnits(ctx.Context()); err != nil {
			ctx.Error(http.StatusInternalServerError, "LoadUnits", err)
			return
		}
		ctx.Permission().Units = ctx.Repository().Units
		ctx.Permission().UnitsMode = make(map[unit.Type]perm.AccessMode)
		for _, u := range ctx.Repository().Units {
			ctx.Permission().UnitsMode[u.Type] = ctx.Permission().AccessMode
		}
	} else {
		permission, err := access_model.GetUserRepoPermissionWithReducer(ctx.Context(), ctx.Repository(), ctx.Doer(), ctx.Reducer())
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetUserRepoPermissionWithReducer", err)
			return
		}
		ctx.SetPermission(&permission)
	}

	if !ctx.Permission().HasAccess() {
		ctx.NotFound()
		return
	}
}
