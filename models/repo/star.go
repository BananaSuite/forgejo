// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// ErrStarNotExist represents a "StarNotExist" kind of error.
type ErrStarNotExist struct {
	UID    int64
	RepoID int64
}

// IsErrStarNotExist checks if an error is a ErrStarNotExist.
func IsErrStarNotExist(err error) bool {
	_, ok := err.(ErrStarNotExist)
	return ok
}

func (err ErrStarNotExist) Error() string {
	return fmt.Sprintf("star does not exist [user_id: %d, repo_id: %d]", err.UID, err.RepoID)
}

func (err ErrStarNotExist) Unwrap() error {
	return util.ErrNotExist
}

// Star represents a starred repo by an user.
type Star struct {
	ID          int64              `xorm:"pk autoincr"`
	UID         int64              `xorm:"UNIQUE(s)"`
	RepoID      int64              `xorm:"UNIQUE(s)"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(Star))
}

// StarRepo or unstar repository.
func StarRepo(ctx context.Context, userID, repoID int64, star bool) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()
	staring := IsStaring(ctx, userID, repoID)

	if star {
		if staring {
			return nil
		}

		if err := db.Insert(ctx, &Star{UID: userID, RepoID: repoID}); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = num_stars + 1 WHERE id = ?", repoID); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `user` SET num_stars = num_stars + 1 WHERE id = ?", userID); err != nil {
			return err
		}
	} else {
		if !staring {
			return nil
		}

		if _, err := db.DeleteByBean(ctx, &Star{UID: userID, RepoID: repoID}); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = num_stars - 1 WHERE id = ?", repoID); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `user` SET num_stars = num_stars - 1 WHERE id = ?", userID); err != nil {
			return err
		}
	}

	return committer.Commit()
}

func GetStar(ctx context.Context, userID, repoID int64) (*Star, error) {
	star := new(Star)
	has, err := db.GetEngine(ctx).Where("uid = ? AND repo_id = ?", userID, repoID).Get(star)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrStarNotExist{UID: userID, RepoID: repoID}
	}
	return star, err
}

// IsStaring checks if user has starred given repository.
func IsStaring(ctx context.Context, userID, repoID int64) bool {
	has, _ := db.GetEngine(ctx).Get(&Star{UID: userID, RepoID: repoID})
	return has
}

// GetStargazers returns the users that starred the repo.
func GetStargazers(ctx context.Context, repo *Repository, opts db.ListOptions) ([]*user_model.User, error) {
	sess := db.GetEngine(ctx).Where("star.repo_id = ?", repo.ID).
		Join("LEFT", "star", "`user`.id = star.uid")
	if opts.Page > 0 {
		sess = db.SetSessionPagination(sess, &opts)

		users := make([]*user_model.User, 0, opts.PageSize)
		return users, sess.Find(&users)
	}

	users := make([]*user_model.User, 0, 8)
	return users, sess.Find(&users)
}

// ClearRepoStars clears all stars for a repository and from the user that starred it.
// Used when a repository is set to private.
func ClearRepoStars(ctx context.Context, repoID int64) error {
	if _, err := db.Exec(ctx, "UPDATE `user` SET num_stars=num_stars-1 WHERE id IN (SELECT `uid` FROM `star` WHERE repo_id = ?)", repoID); err != nil {
		return err
	}

	if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = 0 WHERE id = ?", repoID); err != nil {
		return err
	}

	return db.DeleteBeans(ctx, Star{RepoID: repoID})
}
