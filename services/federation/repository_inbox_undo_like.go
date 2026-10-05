// Copyright 2024, 2025, 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"time"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/repo"
	"forgejo.org/models/user"
	fm "forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"

	ap "github.com/go-ap/activitypub"
)

func ProcessRepositoryInboxUndoLike(ctx context.Context, activity *ap.Activity, repositoryID int64) (ServiceResult, error) {
	undoLikeActivity, err := fm.NewForgeUndoLikeFromActivity(activity)
	if err != nil {
		return ServiceResult{}, NewErrNotAcceptablef("Invalid activity: %v", err)
	}

	log.Trace("Activity validated: %#v", undoLikeActivity)
	user, _, federationHost, err := FindOrCreateFederatedUser(ctx, undoLikeActivity.Actor.GetLink().String())
	if err != nil {
		log.Error("Federated user not found (%s): %v", undoLikeActivity.Actor.GetLink().String(), err)
		return ServiceResult{}, NewErrNotAcceptablef("FindOrCreateFederatedUser: %v", err)
	}

	like, err := undoLikeActivity.Like()
	if err != nil {
		return ServiceResult{}, NewErrNotAcceptablef("Invalid objectId: %v", err)
	}

	objectID, err := fm.NewRepositoryID(like.Object.GetID().String(), string(forgefed.ForgejoSourceType))
	if err != nil {
		return ServiceResult{}, NewErrNotAcceptablef("Parsing repo objectID failed: %v", err)
	}
	log.Trace("Object accepted: %#v", objectID)

	// execute the activity if the repo was already stared
	alreadyStarred := repo.IsStaring(ctx, user.ID, repositoryID)
	if alreadyStarred {
		err = repo.StarRepo(ctx, user.ID, repositoryID, false)
		if err != nil {
			return ServiceResult{}, NewErrNotAcceptablef("Error staring %v", err)
		}
	} else {
		return ServiceResult{}, NewErrNotAcceptablef("star not available: %v", err)
	}

	federationHost.LatestActivity = activity.StartTime
	err = forgefed.UpdateFederationHost(ctx, federationHost)
	if err != nil {
		return ServiceResult{}, NewErrNotAcceptablef("Error updating federatedHost: %v", err)
	}

	return ServiceResult{}, nil
}

func SendUndoLikeActivities(ctx context.Context, doer user.User, repoID int64) error {
	// TODO: The repository APActorID should be fetched from the the database.
	id := repo.APActorID(repoID)
	object, err := fm.NewRepositoryID(id, string(forgefed.ForgejoSourceType))
	if err != nil {
		return err
	}

	activity, err := fm.NewForgeUndoLike(doer.APActorID(), object.AsURI(), time.Now())
	if err != nil {
		return err
	}

	return SendRepositoryActivities(ctx, activity.Activity, doer, repoID)
}
