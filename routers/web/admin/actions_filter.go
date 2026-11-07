// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"slices"
	"strconv"
	"strings"
	"time"

	actions_model "forgejo.org/models/actions"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/timeutil"
	"forgejo.org/services/context"
)

// actionsPeriods are the periods of the time filter, the same ones as on the
// activity page of a repository.
var actionsPeriods = []string{"daily", "halfweekly", "weekly", "monthly", "quarterly", "semiyearly", "yearly"}

// periodStart returns the start of a period that ends now, or the zero time
// for an unknown period (no filter).
func periodStart(period string, now time.Time) time.Time {
	switch period {
	case "daily":
		return now.Add(-24 * time.Hour)
	case "halfweekly":
		return now.Add(-72 * time.Hour)
	case "weekly":
		return now.Add(-168 * time.Hour)
	case "monthly":
		return now.AddDate(0, -1, 0)
	case "quarterly":
		return now.AddDate(0, -3, 0)
	case "semiyearly":
		return now.AddDate(0, -6, 0)
	case "yearly":
		return now.AddDate(-1, 0, 0)
	}
	return time.Time{}
}

type actionsStatusOption struct {
	Status int
	Name   string
}

// actionsFilterChoices are the values offered in the filter dropdowns.
type actionsFilterChoices struct {
	statuses []actions_model.Status
	labels   []string
	repoIDs  []int64
}

// readActionsFilter reads the filters from the query string and puts them,
// with the choices of each dropdown, in ctx.Data for the template. Values
// that are not among the choices are ignored.
func readActionsFilter(ctx *context.Context, statuses []actions_model.Status) actions_model.AdminJobFilter {
	var filter actions_model.AdminJobFilter

	status := actions_model.Status(ctx.FormInt("status"))
	if slices.Contains(statuses, status) {
		filter.Status = status
	}
	filter.Label = ctx.FormTrim("label")
	filter.RepoID = ctx.FormInt64("repo")

	period := ctx.FormTrim("period")
	if start := periodStart(period, time.Now()); !start.IsZero() {
		filter.Since = timeutil.TimeStamp(start.Unix())
	} else {
		period = ""
	}

	ctx.Data["CurStatus"] = int(filter.Status)
	ctx.Data["CurLabel"] = filter.Label
	ctx.Data["CurRepo"] = filter.RepoID
	ctx.Data["CurPeriod"] = period
	return filter
}

// setActionsFilterChoices puts the choices of the filter dropdowns in
// ctx.Data, and adds the filters to the links of the pager.
func setActionsFilterChoices(ctx *context.Context, choices actionsFilterChoices, filter actions_model.AdminJobFilter, pager *context.Pagination) error {
	statusOptions := make([]actionsStatusOption, 0, len(choices.statuses))
	for _, s := range choices.statuses {
		statusOptions = append(statusOptions, actionsStatusOption{Status: int(s), Name: s.LocaleString(ctx.Locale)})
	}

	repos, err := repo_model.GetRepositoriesMapByIDs(ctx, choices.repoIDs)
	if err != nil {
		return err
	}
	repoList := make([]*repo_model.Repository, 0, len(repos))
	for _, repo := range repos {
		repoList = append(repoList, repo)
	}
	slices.SortFunc(repoList, func(a, b *repo_model.Repository) int {
		return strings.Compare(strings.ToLower(a.FullName()), strings.ToLower(b.FullName()))
	})

	ctx.Data["StatusOptions"] = statusOptions
	ctx.Data["LabelOptions"] = choices.labels
	ctx.Data["RepoOptions"] = repoList
	ctx.Data["PeriodOptions"] = actionsPeriods

	if filter.Status != 0 {
		pager.AddParamString("status", strconv.Itoa(int(filter.Status)))
	}
	if filter.Label != "" {
		pager.AddParamString("label", filter.Label)
	}
	if filter.RepoID != 0 {
		pager.AddParamString("repo", strconv.FormatInt(filter.RepoID, 10))
	}
	if period, _ := ctx.Data["CurPeriod"].(string); period != "" {
		pager.AddParamString("period", period)
	}
	return nil
}
