// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import (
	"time"
)

// NotificationThread expose Notification on API
type NotificationThread struct {
	ID         int64                `json:"id"`
	Repository *Repository          `json:"repository"`
	Subject    *NotificationSubject `json:"subject"`
	Release    *Release             `json:"release"`
	Unread     bool                 `json:"unread"`
	Pinned     bool                 `json:"pinned"`
	UpdatedAt  time.Time            `json:"updated_at"`
	URL        string               `json:"url"`
}

// NotificationSubject contains the notification subject (Issue/Pull/Commit)
type NotificationSubject struct {
	Title                string            `json:"title"`
	URL                  string            `json:"url"`
	LatestCommentURL     string            `json:"latest_comment_url"`
	HTMLURL              string            `json:"html_url"`
	LatestCommentHTMLURL string            `json:"latest_comment_html_url"`
	Type                 NotifySubjectType `json:"type" binding:"In(Issue,Pull,Commit,Repository)"`
	State                StateType         `json:"state"`
}

// NotificationCount number of unread notifications
type NotificationCount struct {
	New int64 `json:"new"`
}

// NotifySubjectType represent type of notification subject
type NotifySubjectType string

const (
	// NotifySubjectIssue an issue is subject of a notification
	NotifySubjectIssue NotifySubjectType = "Issue"
	// NotifySubjectPull a pull is subject of a notification
	NotifySubjectPull NotifySubjectType = "Pull"
	// NotifySubjectCommit a commit is subject of a notification
	NotifySubjectCommit NotifySubjectType = "Commit"
	// NotifySubjectRepository a repository is subject of a notification
	NotifySubjectRepository NotifySubjectType = "Repository"
	// NotifySubjectRelease a release is subject of a notification
	NotifySubjectRelease NotifySubjectType = "Release"
)
