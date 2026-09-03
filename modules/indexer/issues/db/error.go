// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package db

import "errors"

var ErrRepoCondAndRepoID = errors.New("Both RepoCond and RepoID have been set, these options are mutually exclusive")
