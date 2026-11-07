// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"context"
	"slices"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/modules/container"
	"forgejo.org/modules/json"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

// RunsOnContainsCond matches the jobs whose runs-on list contains label.
// The runs-on list is stored as a JSON array of strings, so the label is
// searched for as a JSON string, quotes included. LIKE wildcards in the label
// are escaped with "!", which works the same way on every supported database.
func RunsOnContainsCond(column, label string) builder.Cond {
	encoded, err := json.Marshal(label)
	if err != nil {
		// A string always encodes; keep the query safe anyway.
		return builder.Expr("1 = 0")
	}
	escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(string(encoded))
	return builder.Expr(column+" LIKE ? ESCAPE '!'", "%"+escaped+"%")
}

// AdminJobFilter holds the filters of the admin lists of jobs and tasks.
// Zero values mean "no filter".
type AdminJobFilter struct {
	Status Status
	Label  string
	RepoID int64
	Since  timeutil.TimeStamp
}

// ActiveJobStatuses are the statuses of jobs that are not finished.
var ActiveJobStatuses = []Status{StatusWaiting, StatusBlocked, StatusRunning}

// FinishedTaskStatuses are the statuses of tasks that are finished.
var FinishedTaskStatuses = []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped}

// FindActiveJobsOptions finds the jobs of all repositories that are waiting,
// blocked or running, newest first.
type FindActiveJobsOptions struct {
	db.ListOptions
	AdminJobFilter
}

func (opts FindActiveJobsOptions) ToConds() builder.Cond {
	statuses := ActiveJobStatuses
	if slices.Contains(ActiveJobStatuses, opts.Status) {
		statuses = []Status{opts.Status}
	}
	cond := builder.NewCond().And(builder.In("status", statuses))
	if opts.Label != "" {
		cond = cond.And(RunsOnContainsCond("runs_on", opts.Label))
	}
	if opts.RepoID > 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if opts.Since > 0 {
		cond = cond.And(builder.Gte{"created": opts.Since})
	}
	return cond
}

func (opts FindActiveJobsOptions) ToOrders() string {
	return "`id` DESC"
}

// FindFinishedTasksOptions finds the finished tasks of all runners, the most
// recently finished first.
type FindFinishedTasksOptions struct {
	db.ListOptions
	AdminJobFilter
}

func (opts FindFinishedTasksOptions) ToConds() builder.Cond {
	statuses := FinishedTaskStatuses
	if slices.Contains(FinishedTaskStatuses, opts.Status) {
		statuses = []Status{opts.Status}
	}
	cond := builder.NewCond().And(builder.In("status", statuses))
	if opts.Label != "" {
		cond = cond.And(builder.In("job_id",
			builder.Select("id").From("action_run_job").Where(RunsOnContainsCond("runs_on", opts.Label))))
	}
	if opts.RepoID > 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if opts.Since > 0 {
		cond = cond.And(builder.Gte{"stopped": opts.Since})
	}
	return cond
}

func (opts FindFinishedTasksOptions) ToOrders() string {
	return "`stopped` DESC, `id` DESC"
}

// GetActiveJobLabels returns the sorted labels that the waiting, blocked and
// running jobs ask for.
func GetActiveJobLabels(ctx context.Context) ([]string, error) {
	var jobs []*ActionRunJob
	if err := db.GetEngine(ctx).
		In("status", ActiveJobStatuses).
		Cols("runs_on").
		Find(&jobs); err != nil {
		return nil, err
	}
	labels := make(container.Set[string])
	for _, job := range jobs {
		labels.AddMultiple(job.RunsOn...)
	}
	return sortedSet(labels), nil
}

// GetActiveJobRepoIDs returns the IDs of the repositories that have waiting,
// blocked or running jobs.
func GetActiveJobRepoIDs(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := db.GetEngine(ctx).Table("action_run_job").
		In("status", ActiveJobStatuses).
		Distinct("repo_id").
		Find(&ids)
	return ids, err
}

// GetAllRunnerLabels returns the sorted labels of all runners, deleted ones
// included. A task only exists when a runner with the labels of the job
// picked it up, so these are the labels the finished tasks can have.
func GetAllRunnerLabels(ctx context.Context) ([]string, error) {
	var runners []*ActionRunner
	if err := db.GetEngine(ctx).Cols("agent_labels").Unscoped().Find(&runners); err != nil {
		return nil, err
	}
	labels := make(container.Set[string])
	for _, r := range runners {
		labels.AddMultiple(r.AgentLabels...)
	}
	return sortedSet(labels), nil
}

// GetTaskRepoIDs returns the IDs of the repositories that have tasks.
func GetTaskRepoIDs(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := db.GetEngine(ctx).Table("action_task").Distinct("repo_id").Find(&ids)
	return ids, err
}

func sortedSet(set container.Set[string]) []string {
	values := set.Values()
	slices.Sort(values)
	return values
}

// MissingRunnerLabels tells, for each waiting job, the first label that no
// online runner which may run the job has. Jobs that some runner can match
// are left out. Labels with expressions are skipped, as on the actions page
// of a repository.
func MissingRunnerLabels(ctx context.Context, jobs []*ActionRunJob) (map[int64]string, error) {
	missing := make(map[int64]string)
	waiting := slices.DeleteFunc(slices.Clone(jobs), func(job *ActionRunJob) bool {
		return job.Status != StatusWaiting
	})
	if len(waiting) == 0 {
		return missing, nil
	}

	runners, err := db.Find[ActionRunner](ctx, FindRunnerOptions{
		// WithVisible without a repository or owner means: runners of all scopes.
		IsOnline:    optional.Some(true),
		WithVisible: true,
	})
	if err != nil {
		return nil, err
	}

	for _, job := range waiting {
		labels := make(container.Set[string])
		for _, r := range runners {
			global := r.RepoID == 0 && r.OwnerID == 0
			if global || (r.RepoID != 0 && r.RepoID == job.RepoID) || (r.OwnerID != 0 && r.OwnerID == job.OwnerID) {
				labels.AddMultiple(r.AgentLabels...)
			}
		}
		for _, label := range job.RunsOn {
			if strings.Contains(label, "${{") {
				continue
			}
			if !labels.Contains(label) {
				missing[job.ID] = label
				break
			}
		}
	}
	return missing, nil
}
