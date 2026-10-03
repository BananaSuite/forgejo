// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	res, err := validation.IsValid(sut)
	require.NoError(t, err)
	assert.True(t, res, "sut should be valid")

	sut = FollowingRepo{
		ExternalID:       "12",
		FederationHostID: 1,
		URI:              "http://localhost:3000/api/v1/activitypub/repo-id/1",
	}
	res, err = validation.IsValid(sut)
	require.Error(t, err)
	assert.False(t, res, "sut should be invalid")
}
