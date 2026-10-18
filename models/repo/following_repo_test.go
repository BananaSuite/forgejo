// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/assert"
)

func Test_FollowingRepoInbox(t *testing.T) {
	sut := FollowingRepo{URI: "http://localhost:3000/api/v1/activitypub/repo-id/1"}
	assert.Equal(t, "http://localhost:3000/api/v1/activitypub/repo-id/1/inbox", sut.Inbox())
}

func Test_FollowingRepoValidation(t *testing.T) {
	sut := FollowingRepo{
		RepoID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		URI:              "http://localhost:3000/api/v1/activitypub/repo-id/1",
	}
	valid, err := validation.IsValid(sut)
	assert.True(t, valid, "sut '%v' should be valid: %v", sut, err)

	sut = FollowingRepo{
		ExternalID:       "12",
		FederationHostID: 1,
		URI:              "http://localhost:3000/api/v1/activitypub/repo-id/1",
	}
	valid, _ = validation.IsValid(sut)
	assert.False(t, valid, "sut should be invalid")
}
