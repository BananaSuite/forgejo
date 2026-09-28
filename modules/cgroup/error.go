// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package cgroup

import (
	"errors"
	"fmt"
)

var (
	ErrMissingCgroupfsMount      = errors.New("The cgroup2 filesystem is not mounted")
	ErrCgroupMembershipMalformed = errors.New("The cgroup membership information were malformed")
	ErrNotInCgroup               = errors.New("Forgejo is not yet in a cgroup, please start Forgejo in a cgroup which is writable by the user the process is running as")
	ErrCgroupNotDomainType       = errors.New("The cgroup is not of type domain. Please start Forgejo from a cgroup which is a domain cgroup")
	ErrCgroupNotExists           = errors.New("The given cgroup doesn't exist")
)

type ErrMissingControllers struct {
	missing []string
}

func (e ErrMissingControllers) Error() string {
	return fmt.Sprintf("The cgroup is missing the following controllers: %v", e.missing)
}
