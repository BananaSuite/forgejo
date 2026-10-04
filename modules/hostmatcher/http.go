// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package hostmatcher

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"syscall"
	"time"
)

type ErrDialContextHostMatch struct {
	Usage          string
	SettingKeyHint string
	Policy         Policy
	Host           string
	IP             string
}

func (err *ErrDialContextHostMatch) Error() string {
	var addr string
	if err.IP != "" {
		addr = fmt.Sprintf("%s(%s)", err.Host, err.IP)
	} else {
		addr = err.Host
	}

	var prefix string
	switch err.Policy {
	case PolicyAllow:
		prefix = "can only call allowed"
	case PolicyDeny:
		prefix = "can not call blocked"
	}

	return fmt.Sprintf("%s %s HTTP servers (check your %s setting), deny '%s'", err.Usage, prefix, err.SettingKeyHint, addr)
}

// NewDialContext returns a DialContext for Transport, the DialContext will do allow/block list check
func NewDialContext(usage string, allowList, blockList *HostMatchList, proxy *url.URL) func(ctx context.Context, network, addr string) (net.Conn, error) {
	// How Go HTTP Client works with redirection:
	//   transport.RoundTrip URL=http://domain.com, Host=domain.com
	//   transport.DialContext addrOrHost=domain.com:80
	//   dialer.Control tcp4:11.22.33.44:80
	//   transport.RoundTrip URL=http://www.domain.com/, Host=(empty here, in the direction, HTTP client doesn't fill the Host field)
	//   transport.DialContext addrOrHost=domain.com:80
	//   dialer.Control tcp4:11.22.33.44:80
	return func(ctx context.Context, network, addrOrHost string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addrOrHost)
		if err != nil {
			return nil, err
		}

		isSelf := proxy != nil && host == proxy.Hostname() && port == proxy.Port()

		// Block/Allow list by host.
		if !isSelf {
			// if we have an allow-list, check the allow-list first
			if !allowList.IsEmpty() && !allowList.MatchHostName(host) {
				return nil, &ErrDialContextHostMatch{Usage: usage, SettingKeyHint: allowList.SettingKeyHint, Policy: PolicyAllow, Host: host}
			}

			// otherwise, we always follow the blocked list
			if blockList.MatchHostName(host) {
				return nil, &ErrDialContextHostMatch{Usage: usage, SettingKeyHint: blockList.SettingKeyHint, Policy: PolicyDeny, Host: host}
			}
		}

		dialer := net.Dialer{
			// default values comes from http.DefaultTransport
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,

			Control: func(network, ipAddr string, c syscall.RawConn) error {
				// Always allow the host of the proxy, but only on the specified port.
				if isSelf {
					return nil
				}

				// in Control func, the addr was already resolved to IP:PORT format, there is no cost to do ResolveTCPAddr here
				tcpAddr, err := net.ResolveTCPAddr(network, ipAddr)
				if err != nil {
					return fmt.Errorf("%s can only call HTTP servers via TCP, deny '%s(%s:%s)', err=%w", usage, host, network, ipAddr, err)
				}

				// if we have an allow-list, check the allow-list first
				if !allowList.IsEmpty() && !allowList.MatchIPAddr(tcpAddr.IP) {
					return &ErrDialContextHostMatch{Usage: usage, SettingKeyHint: allowList.SettingKeyHint, Policy: PolicyAllow, Host: host, IP: ipAddr}
				}

				// otherwise, we always follow the blocked list
				if blockList.MatchIPAddr(tcpAddr.IP) {
					return &ErrDialContextHostMatch{Usage: usage, SettingKeyHint: blockList.SettingKeyHint, Policy: PolicyDeny, Host: host, IP: ipAddr}
				}

				return nil
			},
		}
		return dialer.DialContext(ctx, network, addrOrHost)
	}
}
