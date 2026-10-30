// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"errors"
	"net/url"
	"strings"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/modules/json"
	"forgejo.org/services/auth/source/oauth2"
	"forgejo.org/services/context"
)

const oidcLogoutSessionKey = "oidc_logout"

// Kept exclusively in the server-side session, never a browser cookie.
type oidcLogoutContext struct {
	SourceID     int64
	UserID       int64
	ClientID     string
	DiscoveryURL string
	Endpoint     string
	IDToken      string
}

func rememberOIDCLogout(ctx *context.Context, source *auth_model.Source, userID int64, idToken string) error {
	if err := ctx.Session.Delete(oidcLogoutSessionKey); err != nil {
		return errors.New("could not clear OIDC logout context")
	}
	config, ok := source.Cfg.(*oauth2.Source)
	if !ok || source.Type != auth_model.OAuth2 || !source.IsActive || idToken == "" || len(idToken) > 16384 || strings.ContainsAny(idToken, "\r\n") {
		return nil
	}
	endpoint := config.OpenIDConnectEndSessionEndpoint()
	if endpoint == "" {
		return nil
	}
	state := oidcLogoutContext{source.ID, userID, config.ClientID, config.OpenIDConnectAutoDiscoveryURL, endpoint, idToken}
	encoded, err := json.Marshal(state)
	if err != nil {
		return errors.New("could not encode OIDC logout context")
	}
	if ctx.Session.Set(oidcLogoutSessionKey, string(encoded)) != nil || ctx.Session.Release() != nil {
		return errors.New("could not persist OIDC logout context")
	}
	return nil
}

func (state *oidcLogoutContext) target(source *auth_model.Source, userID int64) string {
	config, ok := source.Cfg.(*oauth2.Source)
	if !ok || source.Type != auth_model.OAuth2 || !source.IsActive || state.UserID != userID || state.SourceID != source.ID ||
		state.ClientID != config.ClientID || state.DiscoveryURL != config.OpenIDConnectAutoDiscoveryURL ||
		state.IDToken == "" || len(state.IDToken) > 16384 || strings.ContainsAny(state.IDToken, "\r\n") {
		return ""
	}
	endpoint := config.OpenIDConnectEndSessionEndpoint()
	if endpoint == "" || endpoint != state.Endpoint {
		return ""
	}
	target, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	query := target.Query()
	query.Set("client_id", config.ClientID)
	query.Set("id_token_hint", state.IDToken)
	target.RawQuery = query.Encode()
	return target.String()
}

// Capture the redirect before SignOut destroys the native session. Neither
// request parameters nor a local login can supply an ID token for logout.
func oidcLogoutRedirect(ctx *context.Context) string {
	if ctx.Doer == nil {
		return ""
	}
	encoded, ok := ctx.Session.Get(oidcLogoutSessionKey).(string)
	if !ok || len(encoded) > 24576 {
		return ""
	}
	var state oidcLogoutContext
	if json.Unmarshal([]byte(encoded), &state) != nil || state.UserID != ctx.Doer.ID || state.SourceID <= 0 {
		return ""
	}
	source, err := auth_model.GetSourceByID(ctx, state.SourceID)
	if err != nil {
		return ""
	}
	return state.target(source, ctx.Doer.ID)
}
