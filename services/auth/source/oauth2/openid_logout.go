// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package oauth2

import (
	"net/url"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/openidConnect"
)

// OpenIDConnectEndSessionEndpoint returns cached discovery metadata for an
// explicitly enabled source. Logout never performs another discovery request.
func (source *Source) OpenIDConnectEndSessionEndpoint() string {
	if !source.OpenIDConnectEnableLogout || source.Provider != "openidConnect" || source.authSource == nil || !source.authSource.IsActive {
		return ""
	}
	gothRWMutex.RLock()
	defer gothRWMutex.RUnlock()
	registered, err := goth.GetProvider(source.authSource.Name)
	if err != nil {
		return ""
	}
	provider, ok := registered.(*openidConnect.Provider)
	if !ok || provider.ClientKey != source.ClientID || provider.OpenIDConfig == nil {
		return ""
	}
	endpoint := provider.OpenIDConfig.EndSessionEndpoint
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || len(endpoint) > 2048 {
		return ""
	}
	return endpoint
}
