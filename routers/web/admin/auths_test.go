// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"testing"

	"forgejo.org/services/forms"

	"github.com/stretchr/testify/require"
)

func TestParseOAuth2ConfigOpenIDConnectLogout(t *testing.T) {
	for _, provider := range []string{"openidConnect", "github"} {
		for _, enabled := range []bool{false, true} {
			config := parseOAuth2Config(forms.AuthenticationForm{
				Oauth2Provider: provider, OpenIDConnectEnableLogout: enabled,
			})
			require.Equal(t, provider == "openidConnect" && enabled, config.OpenIDConnectEnableLogout)
		}
	}
}
