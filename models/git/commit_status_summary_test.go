package git

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyGetLatestCommitStatusForRepoAndSHAs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ctx := db.DefaultContext
	statuses, err := GetLatestCommitStatusForRepoAndSHAs(ctx, nil)

	require.NoError(t, err)
	assert.Nil(t, statuses)
}
