package federation

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"forgejo.org/models/repo"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/log"

	ap "github.com/go-ap/activitypub"
)

func SendRepositoryActivities(ctx context.Context, activity ap.Activity, doer user.User, repoID int64) error {
	followingRepos, err := repo.FindFollowingReposByRepoID(ctx, repoID)
	log.Trace("Federated Repos is: %#v", followingRepos)
	if err != nil {
		return err
	}

	hosts := make([]*url.URL, 0, len(followingRepos))
	for _, followingRepo := range followingRepos {
		log.Trace("Found following repo: %#v", followingRepo)
		target := followingRepo.URI
		hostURL, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("invalid repository URL: %w", err)
		}
		hosts = append(hosts, hostURL)
	}

	apclientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return err
	}

	apclient, err := apclientFactory.WithKeys(ctx, &doer, doer.KeyID(), hosts)
	if err != nil {
		return err
	}

	for i, repo := range followingRepos {
		activity.StartTime = activity.StartTime.Add(time.Duration(i) * time.Second)
		json, err := activity.MarshalJSON()
		if err != nil {
			return err
		}

		_, err = apclient.Post(json, repo.Inbox())
		if err != nil {
			log.Error("error %v while sending activity: %#v", err, activity)
		}
	}

	return nil
}
