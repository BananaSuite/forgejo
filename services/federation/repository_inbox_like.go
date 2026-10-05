// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"net/http"
	"time"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/repo"
	"forgejo.org/models/user"
	fm "forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	"forgejo.org/modules/validation"
	app_context "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
)

// ProcessLikeActivity receives a ForgeLike activity and does the following:
// Validation of the activity
// Creation of a (remote) federationHost if not existing
// Creation of a forgefed Person if not existing
// Validation of incoming RepositoryID against Local RepositoryID
// Star the repo if it wasn't already stared
// Do some mitigation against out of order attacks
func ProcessLikeActivity(ctx context.Context, activity *ap.Activity, repositoryID int64) (ServiceResult, error) {
	constructorLikeActivity, _ := fm.NewForgeLike(activity.Actor.GetLink().String(), activity.Object.GetLink().String(), activity.StartTime)
	if res, err := validation.IsValid(constructorLikeActivity); !res {
		return ServiceResult{}, NewErrNotAcceptablef("Invalid activity: %v", err)
	}
	log.Trace("Activity validated: %#v", activity)

	// parse actorID (person)
	actorURI := activity.Actor.GetID().String()
	user, _, federationHost, err := FindOrCreateFederatedUser(ctx, actorURI)
	if err != nil {
		log.Error("Federated user not found (%s): %v", actorURI, err)
		return ServiceResult{}, NewErrNotAcceptablef("FindOrCreateFederatedUser failed: %v", err)
	}

	if !constructorLikeActivity.IsNewer(federationHost.LatestActivity) {
		return ServiceResult{}, NewErrNotAcceptablef("LatestActivity: activity already processed: %v", err)
	}

	objectID, err := fm.NewRepositoryID(constructorLikeActivity.Object.GetID().String(), string(forgefed.ForgejoSourceType))
	if err != nil {
		return ServiceResult{}, NewErrNotAcceptablef("Parsing repo objectID failed: %v", err)
	}
	log.Trace("Object accepted: %#v", objectID)

	// execute the activity if the repo was not stared already
	alreadyStarred := repo.IsStaring(ctx, user.ID, repositoryID)
	if !alreadyStarred {
		err = repo.StarRepo(ctx, user.ID, repositoryID, true)
		if err != nil {
			return ServiceResult{}, NewErrNotAcceptablef("Staring failed: %v", err)
		}
	}

	federationHost.LatestActivity = activity.StartTime
	err = forgefed.UpdateFederationHost(ctx, federationHost)
	if err != nil {
		return ServiceResult{}, NewErrNotAcceptablef("Updating federatedHost failed: %v", err)
	}

	return NewServiceResultStatusOnly(http.StatusNoContent), nil
}

// Create or update a list of FollowingRepo structs
func StoreFollowingRepoList(ctx *app_context.Context, localRepoID int64, followingRepoList []string) (int, string, error) {
	followingRepos := make([]*repo.FollowingRepo, 0, len(followingRepoList))
	for _, uri := range followingRepoList {
		federationHost, err := FindOrCreateFederationHost(ctx.Base, uri)
		if err != nil {
			return http.StatusInternalServerError, "Wrong FederationHost", err
		}
		followingRepoID, err := fm.NewRepositoryID(uri, string(federationHost.NodeInfo.SoftwareName))
		if err != nil {
			return http.StatusNotAcceptable, "Invalid federated repo", err
		}
		followingRepo, err := repo.NewFollowingRepo(localRepoID, followingRepoID.ID, federationHost.ID, uri)
		if err != nil {
			return http.StatusNotAcceptable, "Invalid federated repo", err
		}
		followingRepos = append(followingRepos, &followingRepo)
	}

	if err := repo.StoreFollowingRepos(ctx, localRepoID, followingRepos); err != nil {
		return 0, "", err
	}

	return 0, "", nil
}

func DeleteFollowingRepos(ctx context.Context, localRepoID int64) error {
	return repo.StoreFollowingRepos(ctx, localRepoID, []*repo.FollowingRepo{})
}

func SendLikeActivities(ctx context.Context, doer user.User, repoID int64) error {
	// TODO: The repository APActorID should
	id := repo.APActorID(repoID)
	object, err := fm.NewRepositoryID(id, string(forgefed.ForgejoSourceType))
	if err != nil {
		return err
	}

	activity, err := fm.NewForgeLike(doer.APActorID(), object.AsURI(), time.Now())
	if err != nil {
		return err
	}

	return SendRepositoryActivities(ctx, activity.Activity, doer, repoID)
}
