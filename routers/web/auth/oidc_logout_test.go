// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/services/auth/source/oauth2"
	"forgejo.org/services/contexttest"

	"code.forgejo.org/go-chi/session"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/openidConnect"
	"github.com/stretchr/testify/require"
)

func logoutTestSource(t *testing.T) (*auth_model.Source, *openidConnect.Provider) {
	t.Helper()
	config := &oauth2.Source{
		Provider: "openidConnect", ClientID: "test-client", OpenIDConnectEnableLogout: true,
		OpenIDConnectAutoDiscoveryURL: "https://idp.example/.well-known/openid-configuration",
	}
	source := &auth_model.Source{ID: 42, Name: t.Name(), Type: auth_model.OAuth2, IsActive: true, Cfg: config}
	config.SetAuthSource(source)
	provider := &openidConnect.Provider{
		ClientKey:    config.ClientID,
		OpenIDConfig: &openidConnect.OpenIDConfig{EndSessionEndpoint: "https://idp.example/logout?tenant=example"},
	}
	provider.SetName(source.Name)
	goth.UseProviders(provider)
	t.Cleanup(func() { delete(goth.GetProviders(), source.Name) })
	return source, provider
}

func TestOIDCLogoutKeepsTheNativeSignOutAndBindsTheSource(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	source, _ := logoutTestSource(t)
	unittest.AssertSuccessfulInsert(t, source)
	ctx, response := contexttest.MockContext(t, "POST /user/logout")
	ctx.Session = contexttest.NewMockSessionStore("logout-test", "memory", session.Options{})
	ctx.Doer = &user_model.User{ID: 1, Name: "user1"}
	require.NoError(t, ctx.Session.Set("uid", int64(1)))
	require.NoError(t, rememberOIDCLogout(ctx, source, ctx.Doer.ID, "synthetic.id.token"))
	SignOut(ctx)
	require.Equal(t, http.StatusSeeOther, response.Code)
	target, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "idp.example", target.Host)
	require.Equal(t, "example", target.Query().Get("tenant"))
	require.Equal(t, "test-client", target.Query().Get("client_id"))
	require.Equal(t, "synthetic.id.token", target.Query().Get("id_token_hint"))
	require.Nil(t, ctx.Session.Get("uid"))
	require.Nil(t, ctx.Session.Get(oidcLogoutSessionKey))
}

func TestOIDCLogoutRejectsChangedAccountOrConfiguration(t *testing.T) {
	source, provider := logoutTestSource(t)
	config := source.Cfg.(*oauth2.Source)
	state := oidcLogoutContext{
		source.ID, 1, config.ClientID, config.OpenIDConnectAutoDiscoveryURL,
		provider.OpenIDConfig.EndSessionEndpoint, "synthetic.id.token",
	}
	require.NotEmpty(t, state.target(source, 1))
	require.Empty(t, state.target(source, 2))
	for _, change := range []func(){
		func() { source.IsActive = false },
		func() { config.OpenIDConnectEnableLogout = false },
		func() { source.ID++ },
		func() { config.ClientID = "other-client" },
		func() { config.OpenIDConnectAutoDiscoveryURL = "https://other.example/discovery" },
		func() { provider.OpenIDConfig.EndSessionEndpoint = "https://other.example/logout" },
	} {
		source.IsActive, source.ID = true, state.SourceID
		config.OpenIDConnectEnableLogout, config.ClientID = true, state.ClientID
		config.OpenIDConnectAutoDiscoveryURL = state.DiscoveryURL
		provider.OpenIDConfig.EndSessionEndpoint = state.Endpoint
		change()
		require.Empty(t, state.target(source, 1))
	}
}

func TestOIDCLogoutMissingEvidenceUsesLocalLogout(t *testing.T) {
	for _, stored := range []any{nil, "invalid-json", strings.Repeat("x", 24577)} {
		ctx, response := contexttest.MockContext(t, "POST /user/logout?id_token_hint=untrusted")
		ctx.Session = contexttest.NewMockSessionStore("logout-local-test", "memory", session.Options{})
		ctx.Doer = &user_model.User{ID: 1}
		require.NoError(t, ctx.Session.Set(oidcLogoutSessionKey, stored))
		SignOut(ctx)
		require.Equal(t, "/", response.Header().Get("Location"))
		require.Nil(t, ctx.Session.Get(oidcLogoutSessionKey))
	}
	ctx, _ := contexttest.MockContext(t, "/user/logout")
	require.Empty(t, oidcLogoutRedirect(ctx))
}

func TestOIDCLogoutOnlyRecordsUsableOIDCResponses(t *testing.T) {
	source, provider := logoutTestSource(t)
	ctx, _ := contexttest.MockContext(t, "/user/oauth2/test/callback")
	for _, token := range []string{"", strings.Repeat("x", 16385), "synthetic\n.id.token"} {
		require.NoError(t, rememberOIDCLogout(ctx, source, 1, token))
		require.Nil(t, ctx.Session.Get(oidcLogoutSessionKey))
	}
	require.NoError(t, rememberOIDCLogout(ctx, source, 1, "synthetic.id.token"))
	encoded, ok := ctx.Session.Get(oidcLogoutSessionKey).(string)
	require.True(t, ok)
	var state oidcLogoutContext
	require.NoError(t, json.Unmarshal([]byte(encoded), &state))
	require.Equal(t, int64(1), state.UserID)
	provider.OpenIDConfig.EndSessionEndpoint = ""
	require.NoError(t, rememberOIDCLogout(ctx, source, 1, "synthetic.id.token"))
	require.Nil(t, ctx.Session.Get(oidcLogoutSessionKey), "old context is cleared when another login cannot support logout")
}
