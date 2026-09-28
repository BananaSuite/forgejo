// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package cgroup

import (
	"os"
	"path"
	"strings"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

const (
	membershipFile         = "/proc/self/cgroup"
	membershipPrefix       = "0::"
	membershipGroupDeleted = "(deleted)"
)

func ReadCurrentCgroup() (string, error) {
	cgroupRaw, err := os.ReadFile(membershipFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotInCgroup
		}

		return "", err
	}

	strCgroupRaw := string(cgroupRaw)
	if !strings.HasPrefix(strCgroupRaw, membershipPrefix) || !strings.HasSuffix(strCgroupRaw, "\n") {
		return "", ErrCgroupMembershipMalformed
	}

	cgroup := strings.TrimPrefix(strCgroupRaw, membershipPrefix)
	cgroup = strings.TrimSuffix(cgroup, "\n")

	// cgroups cannot be delegated from already deleted groups
	if strings.HasSuffix(cgroup, membershipGroupDeleted) {
		return "", ErrNotInCgroup
	}

	log.Trace("Found cgroup for process: %s", cgroup)
	return path.Join(setting.Cgroup.FsMountpoint, cgroup), nil
}
