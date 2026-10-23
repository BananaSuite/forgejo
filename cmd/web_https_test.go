// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSelfSignedCert(t *testing.T, certFile, keyFile string, mtime time.Time) []byte {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(pemBlockForKey(priv)), 0o600))
	require.NoError(t, os.Chtimes(certFile, mtime, mtime))
	require.NoError(t, os.Chtimes(keyFile, mtime, mtime))

	return der
}

func TestCertReloader(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	now := time.Now()

	derA := writeSelfSignedCert(t, certFile, keyFile, now)
	r := &certReloader{certFile: certFile, keyFile: keyFile}
	require.NoError(t, r.load())

	cert, err := r.GetCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, derA, cert.Certificate[0])

	t.Run("reloads on change", func(t *testing.T) {
		derB := writeSelfSignedCert(t, certFile, keyFile, now.Add(time.Minute))

		cert, err := r.GetCertificate(nil)
		require.NoError(t, err)
		assert.Equal(t, derB, cert.Certificate[0])

		t.Run("keeps previous cert on invalid key", func(t *testing.T) {
			require.NoError(t, os.WriteFile(keyFile, []byte("garbage"), 0o600))
			mtime := now.Add(2 * time.Minute)
			require.NoError(t, os.Chtimes(keyFile, mtime, mtime))

			cert, err := r.GetCertificate(nil)
			require.NoError(t, err)
			assert.Equal(t, derB, cert.Certificate[0])
		})
	})
}
