// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignAndValidate(t *testing.T) {
	cfg, err := setting.NewConfigProviderFromData(`
[actions]
ARTIFACTSV4_SECRET = ForgejoForgejoForgejoForgejoForgejoForgejo8
`)
	require.NoError(t, err)

	defer test.MockVariableValue(&setting.CfgProvider, cfg)()
	setting.LoadCommonSettings()
	require.NoError(t, InitActionsV4())

	endp := "endp"
	expires := "1234"
	artifactName := "valueableArtifact"
	taskID := int64(4711)
	artifactID := int64(815)

	sig1 := buildSignature(endp, expires, artifactName, taskID, artifactID)
	assert.True(t, verifySig(sig1, endp, expires, artifactName, taskID, artifactID))
	badsig := make([]byte, len(sig1))
	copy(badsig, sig1)
	badsig[0]++
	assert.False(t, verifySig(badsig, endp, expires, artifactName, taskID, artifactID))

	cfg, err = setting.NewConfigProviderFromData(`
[actions]
ARTIFACTSV4_SECRET = kmRG31ipa-CRgEWCX0KVUKRbAs8Aa4km3SYQzsYAK9Y
ARTIFACTSV4_KEYS_ACCEPTED = HS256:ForgejoForgejoForgejoForgejoForgejoForgejo8 HS256:GorgejoForgejoForgejoForgejoForgejoForgejo8
`)
	setting.CfgProvider = cfg
	require.NoError(t, err)

	setting.LoadCommonSettings()
	require.NoError(t, InitActionsV4())

	sig2 := buildSignature(endp, expires, artifactName, taskID, artifactID)
	assert.NotEqual(t, sig1, sig2)
	assert.True(t, verifySig(sig1, endp, expires, artifactName, taskID, artifactID))
	assert.True(t, verifySig(sig2, endp, expires, artifactName, taskID, artifactID))
	assert.False(t, verifySig(badsig, endp, expires, artifactName, taskID, artifactID))
}
