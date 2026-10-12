// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package structs

type CreateOrUpdateProjectOptions struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	TemplateType string `json:"template_type"`
	CardType     string `json:"card_type"`
	Status       string `json:"status"`
}

type Project struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	OwnerName    string `json:"owner_name"`
	RepoName     string `json:"repo_name"`
	Status       string `json:"status"`
	OwnerType    string `json:"owner_type"`
	TemplateType string `json:"template_type"`
	CardType     string `json:"card_type"`
}

type MoveProjectIssue struct {
	IssueID int64 `json:"issueID"`
	Sorting int64 `json:"sorting"`
}

type MoveIssuesOption struct {
	ProjectIssues []MoveProjectIssue `json:"issues"`
}

func (m *MoveIssuesOption) GetIssueIDs() []int64 {
	issueIDs := make([]int64, 0, len(m.ProjectIssues))
	for _, issue := range m.ProjectIssues {
		issueIDs = append(issueIDs, issue.IssueID)
	}
	return issueIDs
}

func (m *MoveIssuesOption) GetSortingsMap() map[int64]int64 {
	sortedIssueIDs := make(map[int64]int64)
	for _, issue := range m.ProjectIssues {
		sortedIssueIDs[issue.Sorting] = issue.IssueID
	}
	return sortedIssueIDs
}
