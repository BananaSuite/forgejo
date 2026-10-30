// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package oauth2

import (
	"testing"

	"forgejo.org/models/auth"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/openidConnect"
	"github.com/stretchr/testify/require"
)

func TestOpenIDConnectEndSessionEndpoint(t *testing.T) {
	source := &Source{Provider: "openidConnect", ClientID: "test-client", OpenIDConnectEnableLogout: true}
	source.SetAuthSource(&auth.Source{Name: t.Name(), IsActive: true})
	provider := &openidConnect.Provider{ClientKey: source.ClientID, OpenIDConfig: &openidConnect.OpenIDConfig{}}
	provider.SetName(t.Name())
	goth.UseProviders(provider)
	t.Cleanup(func() { delete(goth.GetProviders(), t.Name()) })
	for _, endpoint := range []string{"", "/logout", "http://idp.example/logout", "javascript:alert(1)", "https://user:password@idp.example/logout", "https://idp.example/logout#fragment", "https://"} {
		provider.OpenIDConfig.EndSessionEndpoint = endpoint
		require.Empty(t, source.OpenIDConnectEndSessionEndpoint())
	}
	provider.OpenIDConfig.EndSessionEndpoint = "https://idp.example/logout?tenant=example"
	require.Equal(t, provider.OpenIDConfig.EndSessionEndpoint, source.OpenIDConnectEndSessionEndpoint())
	source.OpenIDConnectEnableLogout = false
	require.Empty(t, source.OpenIDConnectEndSessionEndpoint())
	source.OpenIDConnectEnableLogout = true
	source.ClientID = "changed-client"
	require.Empty(t, source.OpenIDConnectEndSessionEndpoint())
	source.ClientID = provider.ClientKey
	source.Provider = "github"
	require.Empty(t, source.OpenIDConnectEndSessionEndpoint())
	source.Provider = "openidConnect"
	source.authSource.IsActive = false
	require.Empty(t, source.OpenIDConnectEndSessionEndpoint())
}

func TestOpenIDConnectLogoutConfigurationRoundTrip(t *testing.T) {
	var source Source
	require.NoError(t, source.FromDB([]byte(`{"Provider":"openidConnect"}`)))
	require.False(t, source.OpenIDConnectEnableLogout, "existing sources remain opted out")
	source.OpenIDConnectEnableLogout = true
	encoded, err := source.ToDB()
	require.NoError(t, err)
	var restored Source
	require.NoError(t, restored.FromDB(encoded))
	require.True(t, restored.OpenIDConnectEnableLogout)
}
