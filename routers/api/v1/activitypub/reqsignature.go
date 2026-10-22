// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"net/http"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
	app_context "forgejo.org/services/context"
	"forgejo.org/services/federation"

	"github.com/42wim/httpsig"
)

var FedCtxDataKeyIDKey = "FederationKeyID"

func GetKeyID(ctx app_context.APIContext) (string, error) {
	if !setting.Federation.SignatureEnforced {
		return "", nil
	}

	v := ctx.GetData()[FedCtxDataKeyIDKey]
	if v == nil {
		return "", fmt.Errorf("Error getting keyID from ContextDataStore")
	}

	keyID, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("Error converting value of type (%t) to string", v)
	}

	return keyID, nil
}

func verifyHTTPSignature(ctx app_context.APIContext) (authenticated bool, err error) {
	if !setting.Federation.SignatureEnforced {
		return true, nil
	}

	r := ctx.Req

	// 1. Figure out what key we need to verify
	v, err := httpsig.NewVerifier(r)
	if err != nil {
		log.Debug("For %q verification failed: %v", r.URL.Path, err)
		return false, err
	}

	ctx.Data[FedCtxDataKeyIDKey] = v.KeyId()

	log.Debug("Verify %q, signed by KeyId: %v", r.URL.Path, v.KeyId())
	signatureAlgorithm := httpsig.Algorithm(setting.Federation.SignatureAlgorithms[0])
	pubKey, err := federation.FindOrCreateActorKey(ctx, v.KeyId())
	if err != nil {
		return false, err
	}

	err = v.Verify(pubKey, signatureAlgorithm)
	if err != nil {
		log.Debug("For %q verification failed: %v", r.URL.Path, err)
		return false, err
	}

	err = util.VerifyRequestDigest(r)
	if err != nil {
		log.Debug("For %q digest verification failed: %v", r.URL.Path, err)
		return false, err
	}

	return true, nil
}

// ReqHTTPSignature function
func ReqHTTPSignature() func(ctx *app_context.APIContext) {
	return func(ctx *app_context.APIContext) {
		if authenticated, err := verifyHTTPSignature(*ctx); err != nil {
			log.Warn("verifyHttpSignature failed: %v", err)
			ctx.Error(http.StatusBadRequest, "reqSignature", "request signature verification failed")
		} else if !authenticated {
			ctx.Error(http.StatusForbidden, "reqSignature", "request signature verification failed")
		}
	}
}
