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

	"forgejo.org/modules/test"

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
	defer test.MockVariableValue(&certCheckInterval, 0)()

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

	derB := writeSelfSignedCert(t, certFile, keyFile, now.Add(time.Minute))
	cert, err = r.GetCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, derB, cert.Certificate[0])

	// An invalid key keeps the previous cert and records the error.
	require.NoError(t, os.WriteFile(keyFile, []byte("garbage"), 0o600))
	mtime := now.Add(2 * time.Minute)
	require.NoError(t, os.Chtimes(keyFile, mtime, mtime))
	cert, err = r.GetCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, derB, cert.Certificate[0])
	assert.NotEmpty(t, r.lastErr)

	// A valid pair clears the error.
	derC := writeSelfSignedCert(t, certFile, keyFile, now.Add(3*time.Minute))
	cert, err = r.GetCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, derC, cert.Certificate[0])
	assert.Empty(t, r.lastErr)

	// A rewrite within the same mtime tick is picked up if the size changed.
	oldInfo, err := os.Stat(certFile)
	require.NoError(t, err)
	sameMtime := now.Add(3 * time.Minute)
	derD := writeSelfSignedCert(t, certFile, keyFile, sameMtime)
	if newInfo, err := os.Stat(certFile); err == nil && newInfo.Size() == oldInfo.Size() {
		f, err := os.OpenFile(certFile, os.O_APPEND|os.O_WRONLY, 0)
		require.NoError(t, err)
		_, err = f.WriteString("\n")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		require.NoError(t, os.Chtimes(certFile, sameMtime, sameMtime))
	}
	cert, err = r.GetCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, derD, cert.Certificate[0])

	// Changes within the check interval are not picked up.
	defer test.MockVariableValue(&certCheckInterval, time.Hour)()
	writeSelfSignedCert(t, certFile, keyFile, now.Add(4*time.Minute))
	cert, err = r.GetCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, derD, cert.Certificate[0])
}
