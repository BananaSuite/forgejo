// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package cgroup

import (
	"fmt"
	"os"
	"sync/atomic"

	"forgejo.org/modules/cgroup"
	"forgejo.org/modules/container"
	"forgejo.org/modules/setting"
)

var (
	RootManager    atomic.Pointer[cgroup.Manager]
	GitProcManager atomic.Pointer[cgroup.Manager]
)

func Init() error {
	if !setting.Cgroup.Enabled {
		return nil
	}

	currentCgroup, err := cgroup.ReadCurrentCgroup()
	if err != nil {
		return err
	}

	rootManager, err := cgroup.LoadManager(currentCgroup)
	if err != nil {
		return err
	}

	RootManager.Store(rootManager)

	mainProcManager, err := cgroup.NewManager("forgejo")
	if err != nil {
		return err
	}

	gitProcManager, err := cgroup.NewManager("git")
	if err != nil {
		return err
	}

	GitProcManager.Store(gitProcManager)

	pid := os.Getpid()
	err = mainProcManager.AddProc(pid)
	if err != nil {
		return err
	}

	err = rootManager.SetSubtreeControllers(container.SetOf("pids"))
	if err != nil {
		return err
	}

	err = gitProcManager.PidSetMax(fmt.Sprintf("%d", setting.Cgroup.PidLimit))
	if err != nil {
		return err
	}

	return err
}
