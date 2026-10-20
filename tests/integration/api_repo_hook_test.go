// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	webhook_model "forgejo.org/models/webhook"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPICreateHook(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 37})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// user1 is an admin user
	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/%s", owner.Name, repo.Name, "hooks"), api.CreateHookOption{
		Type: "gitea",
		Config: api.CreateHookOptionConfig{
			"content_type": "json",
			"url":          "http://example.com/",
		},
		AuthorizationHeader: "Bearer s3cr3t",
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var apiHook *api.Hook
	DecodeJSON(t, resp, &apiHook)
	assert.Equal(t, "http://example.com/", apiHook.Config["url"])
	assert.Equal(t, "http://example.com/", apiHook.URL)
	assert.Equal(t, "Bearer s3cr3t", apiHook.AuthorizationHeader)
}

func TestApiUpdateHook(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 37})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	createReq := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/hooks", owner.Name, repo.Name), api.CreateHookOption{
		Type: "forgejo",
		Config: api.CreateHookOptionConfig{
			"content_type": "json",
			"url":          "http://example.com/",
			"secret":       "secret1",
		},
		AuthorizationHeader: "Bearer s3cr3t",
	}).AddTokenAuth(token)
	createResp := MakeRequest(t, createReq, http.StatusCreated)

	var apiHookCreate *api.Hook
	DecodeJSON(t, createResp, &apiHookCreate)

	req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/hooks/%d", owner.Name, repo.Name, apiHookCreate.ID), api.EditHookOption{
		Config: map[string]string{
			"content_type": "json",
			"url":          "http://example.com/updated",
			"secret":       "secret2",
		},
		AuthorizationHeader: "Bearer updated",
	}).AddTokenAuth(token)
	updateResp := MakeRequest(t, req, http.StatusOK)

	var apiHookUpdate *api.Hook
	DecodeJSON(t, updateResp, &apiHookUpdate)
	assert.Equal(t, "http://example.com/updated", apiHookUpdate.Config["url"])
	assert.Equal(t, "Bearer updated", apiHookUpdate.AuthorizationHeader)
	assert.NotContains(t, apiHookUpdate.Config, "secret")
	webhook := unittest.AssertExistsAndLoadBean(t, &webhook_model.Webhook{ID: apiHookCreate.ID})
	assert.Equal(t, "secret2", webhook.Secret)
}
