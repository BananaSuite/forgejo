// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/services/contexttest"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{})
}

func TestVerifyKeyIDMatchesActorID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, true)()
	defer test.MockVariableValue(&setting.Federation.InsecureAllowInvalidHosts, true)()
	defer test.MockVariableValue(&setting.Federation.MaxSize, 2048)()
	defer test.MockVariableValue(&setting.Federation.PostHeaders, []string{"(request-target)", "Date", "Host", "Digest"})()

	ctx, _ := contexttest.MockAPIContext(t, "/api/v1/activitypub/user-id/2/inbox")
	mock := test.NewFederationServerMock()
	federatedSrvURL := "https://127.0.0.1:12345"

	actor := ap.Actor{ID: ap.IRI(mock.Persons[0].ActorID(federatedSrvURL))}
	activity := ap.Activity{Actor: actor}

	t.Run("valid_signed_with_user_key", func(t *testing.T) {
		ctx.Data[FedCtxDataKeyIDKey] = mock.Persons[0].KeyID(federatedSrvURL)

		err := VerifyKeyIDMatchesActorID(ctx, &activity)
		require.NoError(t, err)
	})

	t.Run("valid_signed_with_host_key", func(t *testing.T) {
		ctx.Data[FedCtxDataKeyIDKey] = mock.ApActor.KeyID(federatedSrvURL)

		err := VerifyKeyIDMatchesActorID(ctx, &activity)
		require.NoError(t, err)
	})

	t.Run("invalid_request_false_host", func(t *testing.T) {
		ctx.Data[FedCtxDataKeyIDKey] = mock.ApActor.KeyID("https://some-attacker.org")

		err := VerifyKeyIDMatchesActorID(ctx, &activity)
		require.Error(t, err)
	})

	t.Run("invalid_request_false_keyid", func(t *testing.T) {
		ctx.Data[FedCtxDataKeyIDKey] = mock.Persons[1].KeyID(federatedSrvURL)

		err := VerifyKeyIDMatchesActorID(ctx, &activity)
		require.Error(t, err)
	})
}
