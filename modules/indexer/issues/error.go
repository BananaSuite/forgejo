// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package issues

import "errors"

var ErrRepoCondNotDb = errors.New("RepoCond set on a non-db indexer type, only RepoIDs are supported by non-db indexers")
