// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/container"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

type TaskList []*ActionTask

func (tasks TaskList) GetJobIDs() []int64 {
	return container.FilterSlice(tasks, func(t *ActionTask) (int64, bool) {
		return t.JobID, t.JobID != 0
	})
}

func (tasks TaskList) LoadJobs(ctx context.Context) error {
	jobIDs := tasks.GetJobIDs()
	jobs := make(map[int64]*ActionRunJob, len(jobIDs))
	if err := db.GetEngine(ctx).In("id", jobIDs).Find(&jobs); err != nil {
		return err
	}
	for _, t := range tasks {
		if t.JobID > 0 && t.Job == nil {
			t.Job = jobs[t.JobID]
		}
	}

	// TODO: Replace with "ActionJobList(maps.Values(jobs))" once available
	var jobsList ActionJobList = make([]*ActionRunJob, 0, len(jobs))
	for _, j := range jobs {
		jobsList = append(jobsList, j)
	}
	return jobsList.LoadAttributes(ctx, true)
}

// GetRunnerIDs returns the distinct IDs of the runners that picked up the tasks.
func (tasks TaskList) GetRunnerIDs() []int64 {
	return container.FilterSlice(tasks, func(t *ActionTask) (int64, bool) {
		return t.RunnerID, t.RunnerID != 0
	})
}

// LoadRunners returns the runners of the tasks, keyed by runner ID. Deleted
// runners are included, so that old tasks still show which runner ran them.
func (tasks TaskList) LoadRunners(ctx context.Context) (map[int64]*ActionRunner, error) {
	runnerIDs := tasks.GetRunnerIDs()
	runners := make(map[int64]*ActionRunner, len(runnerIDs))
	if len(runnerIDs) == 0 {
		return runners, nil
	}
	if err := db.GetEngine(ctx).In("id", runnerIDs).Unscoped().Find(&runners); err != nil {
		return nil, err
	}
	return runners, nil
}

func (tasks TaskList) LoadAttributes(ctx context.Context) error {
	return tasks.LoadJobs(ctx)
}

type FindTaskOptions struct {
	db.ListOptions
	RepoID        int64
	OwnerID       int64
	CommitSHA     string
	Status        []Status
	UpdatedBefore timeutil.TimeStamp
	StartedBefore timeutil.TimeStamp
	RunnerID      int64
	LogExpired    optional.Option[bool]
	LogInStorage  optional.Option[bool]
}

func (opts FindTaskOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RepoID > 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if opts.OwnerID != 0 {
		cond = cond.And(builder.Eq{"owner_id": opts.OwnerID})
	}
	if opts.CommitSHA != "" {
		cond = cond.And(builder.Eq{"commit_sha": opts.CommitSHA})
	}
	if len(opts.Status) > 0 {
		cond = cond.And(builder.In("status", opts.Status))
	}
	if opts.UpdatedBefore > 0 {
		cond = cond.And(builder.Lt{"updated": opts.UpdatedBefore})
	}
	if opts.StartedBefore > 0 {
		cond = cond.And(builder.Lt{"started": opts.StartedBefore})
	}
	if opts.RunnerID > 0 {
		cond = cond.And(builder.Eq{"runner_id": opts.RunnerID})
	}
	if has, value := opts.LogExpired.Get(); has {
		cond = cond.And(builder.Eq{"log_expired": value})
	}
	if has, value := opts.LogInStorage.Get(); has {
		cond = cond.And(builder.Eq{"log_in_storage": value})
	}
	return cond
}

func (opts FindTaskOptions) ToOrders() string {
	return "`id` DESC"
}
