// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/services/auth/source/oauth2"
	"forgejo.org/tests"

	"github.com/stretchr/testify/require"
)

func TestAdminAuthOpenIDConnectLogout(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")
	response := session.MakeRequest(t, NewRequest(t, "GET", "/admin/auths/new"), http.StatusOK)
	document := NewHTMLParser(t, response.Body)
	document.AssertElement(t, "#open_id_connect_enable_logout", true)
	document.AssertElement(t, "#open_id_connect_enable_logout[checked]", false)

	// Keep the source inactive: this tests the real settings and persistence
	// paths without contacting an external discovery endpoint.
	payload := map[string]string{
		"type": fmt.Sprintf("%d", auth.OAuth2), "name": "oidc-with-logout",
		"oauth2_provider": "openidConnect", "oauth2_key": "test-client", "oauth2_secret": "test-secret",
		"open_id_connect_auto_discovery_url": "https://idp.example/.well-known/openid-configuration",
		"open_id_connect_enable_logout":      "on",
	}
	source := addAuthSource(t, payload)
	require.True(t, source.Cfg.(*oauth2.Source).OpenIDConnectEnableLogout)
	path := fmt.Sprintf("/admin/auths/%d", source.ID)
	for _, enabled := range []bool{true, false, true} {
		if enabled {
			payload["open_id_connect_enable_logout"] = "on"
		} else {
			delete(payload, "open_id_connect_enable_logout")
		}
		session.MakeRequest(t, NewRequestWithValues(t, "POST", path, payload), http.StatusSeeOther)
		stored, err := auth.GetSourceByID(db.DefaultContext, source.ID)
		require.NoError(t, err)
		require.Equal(t, enabled, stored.Cfg.(*oauth2.Source).OpenIDConnectEnableLogout)
		response = session.MakeRequest(t, NewRequest(t, "GET", path), http.StatusOK)
		document = NewHTMLParser(t, response.Body)
		document.AssertElement(t, "#open_id_connect_enable_logout[checked]", enabled)
	}
}
