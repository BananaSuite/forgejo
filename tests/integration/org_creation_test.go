// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestOrgCreatePublic(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session1 := loginUser(t, "user2")
	session1.MakeRequest(t, NewRequestWithValues(t, "POST", "/org/create", map[string]string{
		"org_name":   "my_test_org",
		"visibility": "0",
	}), http.StatusSeeOther)

	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{LowerName: "my_test_org"})
	assert.Equal(t, structs.VisibleTypePublic, org.Visibility)
	assert.True(t, org.VisibleMembers)
}

func TestOrgCreatePrivate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session1 := loginUser(t, "user2")
	session1.MakeRequest(t, NewRequestWithValues(t, "POST", "/org/create", map[string]string{
		"org_name":   "my_test_org",
		"visibility": "2",
	}), http.StatusSeeOther)

	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{LowerName: "my_test_org"})
	assert.Equal(t, structs.VisibleTypePrivate, org.Visibility)
	assert.False(t, org.VisibleMembers)
}
