// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package doctor

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

func init() {
	Register(&Check{
		Title:                      "git fsck",
		Name:                       "git-fsck",
		IsDefault:                  false,
		Run:                        gitFsck,
		AbortIfFailed:              false,
		SkipDatabaseInitialization: false, // git.InitFull happens in initDBSkipLogger
		Priority:                   10,
	})
}

func gitFsck(ctx context.Context, logger log.Logger, autofix bool) error {
	paths, err := filepath.Glob(setting.RepoRootPath + "/*/*.git")
	if err != nil {
		return err
	}
	cmd := git.NewCommand(ctx, "fsck", "--full", "--no-dangling")
	logger.Info("%d repositories to check with: %s", len(paths), cmd.String())
	previous := time.Now()
	broken := 0
	for i, p := range paths {
		now := time.Now()
		if now.Sub(previous) > 3*time.Second {
			previous = now
			logger.Info("%d/%d checked", i, len(paths))
		}
		err := cmd.Run(&git.RunOpts{
			Dir: p,
		})
		if err != nil {
			logger.Warn("%s is broken: %v", p, err)
			broken++
		}
	}
	if broken > 0 {
		return fmt.Errorf("%d repositories are broken", broken)
	}
	return nil
}
