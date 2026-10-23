// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"crypto/tls"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

var tlsVersionStringMap = map[string]uint16{
	"tlsv1.0": tls.VersionTLS10,
	"tlsv1.1": tls.VersionTLS11,
	"tlsv1.2": tls.VersionTLS12,
	"tlsv1.3": tls.VersionTLS13,
}

func toTLSVersion(version string) uint16 {
	tlsVersion, ok := tlsVersionStringMap[strings.TrimSpace(strings.ToLower(version))]
	if !ok {
		log.Warn("Unknown tls version: %s", version)
		return 0
	}
	return tlsVersion
}

var curveStringMap = map[string]tls.CurveID{
	"x25519":             tls.X25519,
	"p256":               tls.CurveP256,
	"p384":               tls.CurveP384,
	"p521":               tls.CurveP521,
	"x25519mlkem768":     tls.X25519MLKEM768,
	"secp256r1mlkem768":  tls.SecP256r1MLKEM768,
	"secp384r1mlkem1024": tls.SecP384r1MLKEM1024,
}

func toCurvePreferences(preferences []string) []tls.CurveID {
	ids := make([]tls.CurveID, 0, len(preferences))
	for _, pref := range preferences {
		id, ok := curveStringMap[strings.TrimSpace(strings.ToLower(pref))]
		if !ok {
			log.Warn("Unknown curve: %s", pref)
		}
		if id != 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

var cipherStringMap = map[string]uint16{
	"rsa_with_rc4_128_sha":                      tls.TLS_RSA_WITH_RC4_128_SHA,
	"rsa_with_3des_ede_cbc_sha":                 tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	"rsa_with_aes_128_cbc_sha":                  tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"rsa_with_aes_256_cbc_sha":                  tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"rsa_with_aes_128_cbc_sha256":               tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	"rsa_with_aes_128_gcm_sha256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	"rsa_with_aes_256_gcm_sha384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	"ecdhe_ecdsa_with_rc4_128_sha":              tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
	"ecdhe_ecdsa_with_aes_128_cbc_sha":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"ecdhe_ecdsa_with_aes_256_cbc_sha":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"ecdhe_rsa_with_rc4_128_sha":                tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	"ecdhe_rsa_with_3des_ede_cbc_sha":           tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	"ecdhe_rsa_with_aes_128_cbc_sha":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"ecdhe_rsa_with_aes_256_cbc_sha":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"ecdhe_ecdsa_with_aes_128_cbc_sha256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	"ecdhe_rsa_with_aes_128_cbc_sha256":         tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	"ecdhe_rsa_with_aes_128_gcm_sha256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"ecdhe_ecdsa_with_aes_128_gcm_sha256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"ecdhe_rsa_with_aes_256_gcm_sha384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"ecdhe_ecdsa_with_aes_256_gcm_sha384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"ecdhe_rsa_with_chacha20_poly1305_sha256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	"ecdhe_ecdsa_with_chacha20_poly1305_sha256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	"ecdhe_rsa_with_chacha20_poly1305":          tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	"ecdhe_ecdsa_with_chacha20_poly1305":        tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	"aes_128_gcm_sha256":                        tls.TLS_AES_128_GCM_SHA256,
	"aes_256_gcm_sha384":                        tls.TLS_AES_256_GCM_SHA384,
	"chacha20_poly1305_sha256":                  tls.TLS_CHACHA20_POLY1305_SHA256,
}

func toTLSCiphers(cipherStrings []string) []uint16 {
	ciphers := make([]uint16, 0, len(cipherStrings))
	for _, cipherString := range cipherStrings {
		cipher, ok := cipherStringMap[strings.TrimSpace(strings.ToLower(cipherString))]
		if !ok {
			log.Warn("Unknown cipher: %s", cipherString)
		}
		if cipher != 0 {
			ciphers = append(ciphers, cipher)
		}
	}

	return ciphers
}

// runHTTPS listens on the provided network address and then calls
// Serve to handle requests on incoming TLS connections.
//
// Filenames containing a certificate and matching private key for the server must
// be provided. If the certificate is signed by a certificate authority, the
// certFile should be the concatenation of the server's certificate followed by the
// CA's certificate.
func runHTTPS(network, listenAddr, name, certFile, keyFile string, m http.Handler, useProxyProtocol, proxyProtocolTLSBridging bool) error {
	tlsConfig := &tls.Config{}
	if tlsConfig.NextProtos == nil {
		tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	}

	if version := toTLSVersion(setting.SSLMinimumVersion); version != 0 {
		tlsConfig.MinVersion = version
	}
	if version := toTLSVersion(setting.SSLMaximumVersion); version != 0 {
		tlsConfig.MaxVersion = version
	}

	if curves := toCurvePreferences(setting.SSLCurvePreferences); len(curves) > 0 {
		tlsConfig.CurvePreferences = curves
	}

	if ciphers := toTLSCiphers(setting.SSLCipherSuites); len(ciphers) > 0 {
		tlsConfig.CipherSuites = ciphers
	}

	r := &certReloader{certFile: certFile, keyFile: keyFile}
	if err := r.load(); err != nil {
		log.Error("Failed to load https cert file %s and key file %s for %s:%s: %v", certFile, keyFile, network, listenAddr, err)
		return err
	}
	tlsConfig.GetCertificate = r.GetCertificate

	return graceful.HTTPListenAndServeTLSConfig(network, listenAddr, name, tlsConfig, m, useProxyProtocol, proxyProtocolTLSBridging)
}

func runHTTPSWithTLSConfig(network, listenAddr, name string, tlsConfig *tls.Config, m http.Handler, useProxyProtocol, proxyProtocolTLSBridging bool) error {
	return graceful.HTTPListenAndServeTLSConfig(network, listenAddr, name, tlsConfig, m, useProxyProtocol, proxyProtocolTLSBridging)
}

// certCheckInterval limits how often handshakes check the certificate files for changes.
var certCheckInterval = time.Second

// certReloader serves the certificate from certFile and keyFile and reloads it
// when either file's modification time changes.
type certReloader struct {
	certFile, keyFile string

	mu              sync.Mutex
	cert            *tls.Certificate
	certMod, keyMod time.Time
	checked         time.Time
	lastErr         string
}

// load must be called with r.mu held once the listener is serving.
func (r *certReloader) load() error {
	certInfo, err := os.Stat(r.certFile)
	if err != nil {
		return err
	}
	keyInfo, err := os.Stat(r.keyFile)
	if err != nil {
		return err
	}

	if r.cert != nil && certInfo.ModTime().Equal(r.certMod) && keyInfo.ModTime().Equal(r.keyMod) {
		return nil
	}

	cert, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return err
	}
	if r.cert != nil {
		log.Info("Reloaded https cert file %s and key file %s", r.certFile, r.keyFile)
	}
	r.cert, r.certMod, r.keyMod = &cert, certInfo.ModTime(), keyInfo.ModTime()
	return nil
}

// GetCertificate keeps serving the previous certificate if a reload fails, which
// covers a renewal that has written the cert file but not yet the key file.
func (r *certReloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if time.Since(r.checked) < certCheckInterval {
		return r.cert, nil
	}
	r.checked = time.Now()

	if err := r.load(); err == nil {
		r.lastErr = ""
	} else if err.Error() != r.lastErr {
		r.lastErr = err.Error()
		log.Error("Failed to reload https cert file %s and key file %s: %v", r.certFile, r.keyFile, err)
	}
	return r.cert, nil
}
