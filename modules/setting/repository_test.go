// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"fmt"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func loadRepositoryFromData(t *testing.T, config string) error {
	cfg, err := NewConfigProviderFromData(config)
	require.NoError(t, err)
	return loadRepositoryFrom(cfg)
}

func TestSettingRepositoryMapping(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		require.NoError(t, loadRepositoryFromData(t, ""))
		assert.Equal(t, path.Join(AppDataPath, "forgejo-repositories"), Repository.RootPath)
		assert.Equal(t, "bash", ScriptType)
		assert.Equal(t, []string{"UTF-8", "UTF-16BE", "UTF-16LE", "UTF-32BE", "UTF-32LE", "ISO-8859-1", "windows-1252", "ISO-8859-2", "windows-1250", "ISO-8859-5", "ISO-8859-6", "ISO-8859-7", "windows-1253", "ISO-8859-8-I", "windows-1255", "ISO-8859-8", "windows-1251", "windows-1256", "KOI8-R", "ISO-8859-9", "windows-1254", "Shift_JIS", "GB18030", "EUC-JP", "EUC-KR", "Big5", "ISO-2022-JP", "ISO-2022-KR", "ISO-2022-CN", "IBM424_rtl", "IBM424_ltr", "IBM420_rtl", "IBM420_ltr"}, Repository.DetectedCharsetsOrder)
		assert.Equal(t, map[string]int{"big5": 25, "euc-jp": 23, "euc-kr": 24, "gb18030": 22, "ibm420_ltr": 32, "ibm420_rtl": 31, "ibm424_ltr": 30, "ibm424_rtl": 29, "iso-2022-cn": 28, "iso-2022-jp": 26, "iso-2022-kr": 27, "iso-8859-1": 5, "iso-8859-2": 7, "iso-8859-5": 9, "iso-8859-6": 10, "iso-8859-7": 11, "iso-8859-8": 15, "iso-8859-8-i": 13, "iso-8859-9": 19, "koi8-r": 18, "shift_jis": 21, "utf-16be": 1, "utf-16le": 2, "utf-32be": 3, "utf-32le": 4, "utf-8": 0, "windows-1250": 8, "windows-1251": 16, "windows-1252": 6, "windows-1253": 12, "windows-1254": 20, "windows-1255": 14, "windows-1256": 17}, Repository.DetectedCharsetScore)
		assert.Empty(t, Repository.AnsiCharset)
		assert.IsType(t, "", Repository.AnsiCharset)
		assert.False(t, Repository.ForcePrivate)
		assert.Equal(t, "last", Repository.DefaultPrivate)
		assert.True(t, Repository.DefaultPushCreatePrivate)
		assert.Equal(t, -1, Repository.MaxCreationLimit)
		assert.Equal(t, []string{"Apache-2.0", "MIT"}, Repository.PreferredLicenses)
		assert.False(t, Repository.DisableHTTPGit)
		assert.Empty(t, Repository.AccessControlAllowOrigin)
		assert.IsType(t, "", Repository.AccessControlAllowOrigin)
		assert.True(t, Repository.UseCompatSSHURI)
		assert.Equal(t, "https", Repository.GoGetCloneURLProtocol)
		assert.False(t, Repository.DefaultCloseIssuesViaCommitsInAnyBranch)
		assert.False(t, Repository.EnablePushCreateUser)
		assert.False(t, Repository.EnablePushCreateOrg)
		assert.Equal(t, []string{}, Repository.DisabledRepoUnits)
		assert.Equal(t, []string{}, Repository.DefaultRepoUnits)
		assert.Equal(t, []string{}, Repository.DefaultForkRepoUnits)
		assert.Equal(t, []string{}, Repository.DefaultMirrorRepoUnits)
		assert.True(t, Repository.PrefixArchiveFiles)
		assert.False(t, Repository.DisableMigrations)
		assert.False(t, Repository.DisableStars)
		assert.False(t, Repository.DisableForks)
		assert.Equal(t, "main", Repository.DefaultBranch)
		assert.False(t, Repository.AllowAdoptionOfUnadoptedRepositories)
		assert.False(t, Repository.AllowDeleteOfUnadoptedRepositories)
		assert.False(t, Repository.DisableDownloadSourceArchives)
		assert.True(t, Repository.AllowForkWithoutMaximumLimit)
		assert.Equal(t, []string{".txt", ".md", ".markdown", ".mdown", ".mkd", ".livemd", ""}, Repository.Editor.LineWrapExtensions)
		assert.True(t, Repository.Upload.Enabled)
		assert.Equal(t, path.Join(AppWorkPath, "data/tmp/uploads"), Repository.Upload.TempPath)
		assert.Empty(t, Repository.Upload.AllowedTypes)
		assert.IsType(t, "", Repository.Upload.AllowedTypes)
		assert.Equal(t, int64(50), Repository.Upload.FileMaxSize)
		assert.Equal(t, 5, Repository.Upload.MaxFiles)
		assert.Equal(t, "tmp/local-repo", Repository.Local.LocalCopyPath)
		assert.Equal(t, []string{"WIP:", "[WIP]"}, Repository.PullRequest.WorkInProgressPrefixes)
		assert.Equal(t, []string{"close", "closes", "closed", "fix", "fixes", "fixed", "resolve", "resolves", "resolved"}, Repository.PullRequest.CloseKeywords)
		assert.Equal(t, []string{"reopen", "reopens", "reopened"}, Repository.PullRequest.ReopenKeywords)
		assert.Equal(t, "merge", Repository.PullRequest.DefaultMergeStyle)
		assert.Equal(t, 50, Repository.PullRequest.DefaultMergeMessageCommitsLimit)
		assert.Equal(t, 5120, Repository.PullRequest.DefaultMergeMessageSize)
		assert.False(t, Repository.PullRequest.DefaultMergeMessageAllAuthors)
		assert.Equal(t, 10, Repository.PullRequest.DefaultMergeMessageMaxApprovers)
		assert.True(t, Repository.PullRequest.DefaultMergeMessageOfficialApproversOnly)
		assert.Equal(t, "merge", Repository.PullRequest.DefaultUpdateStyle)
		assert.False(t, Repository.PullRequest.PopulateSquashCommentWithCommitMessages)
		assert.True(t, Repository.PullRequest.RetargetChildrenOnMerge)
		assert.True(t, Repository.PullRequest.KeepForcePushRefs)
		assert.Equal(t, []string{"Too heated", "Off-topic", "Spam", "Resolved"}, Repository.Issue.LockReasons)
		assert.Equal(t, 3, Repository.Issue.MaxPinned)
		assert.Empty(t, Repository.Release.AllowedTypes)
		assert.IsType(t, "", Repository.Release.AllowedTypes)
		assert.Equal(t, 10, Repository.Release.DefaultPagingNum)
		assert.Equal(t, "default", Repository.Signing.SigningKey)
		assert.Empty(t, Repository.Signing.SigningName)
		assert.IsType(t, "", Repository.Signing.SigningName)
		assert.Empty(t, Repository.Signing.SigningEmail)
		assert.IsType(t, "", Repository.Signing.SigningEmail)
		assert.Equal(t, "openpgp", Repository.Signing.Format)
		assert.Equal(t, []string{"always"}, Repository.Signing.InitialCommit)
		assert.Equal(t, []string{"pubkey", "twofa", "parentsigned"}, Repository.Signing.CRUDActions)
		assert.Equal(t, []string{"pubkey", "twofa", "basesigned", "commitssigned"}, Repository.Signing.Merges)
		assert.Equal(t, []string{"never"}, Repository.Signing.Wiki)
		assert.Equal(t, "collaborator", Repository.Signing.DefaultTrustModel)
		assert.Equal(t, []string(nil), Repository.SettableFlags)
		assert.False(t, Repository.EnableFlags)
		assert.Equal(t, 36, MaxUserCardsPerPage)
		assert.Equal(t, 40, MaxForksPerPage)
		assert.Nil(t, SSHInstanceKey)
	})

	t.Run("Specified", func(t *testing.T) {
		require.NoError(t, loadRepositoryFromData(t, `
[repository]
SCRIPT_TYPE = sh
FORCE_PRIVATE = true
[repository.editor]
LINE_WRAP_EXTENSIONS = .txt
[repository.upload]
ENABLED = false
[repository.local]
LOCAL_COPY_PATH = tmp/repo-local
[repository.pull-request]
WORK_IN_PROGRESS_PREFIXES = wip:,draft:
[repository.issue]
MAX_PINNED = 5
[repository.release]
ALLOWED_TYPES = .zip,text/plain
[repository.signing]
SIGNING_KEY = none
`))
		assert.Equal(t, "sh", ScriptType)
		assert.True(t, Repository.ForcePrivate)
		assert.Equal(t, []string{".txt"}, Repository.Editor.LineWrapExtensions)
		assert.False(t, Repository.Upload.Enabled)
		assert.Equal(t, "tmp/repo-local", Repository.Local.LocalCopyPath)
		assert.Equal(t, []string{"wip:", "draft:"}, Repository.PullRequest.WorkInProgressPrefixes)
		assert.Equal(t, 5, Repository.Issue.MaxPinned)
		assert.Equal(t, ".zip,text/plain", Repository.Release.AllowedTypes)
		assert.Equal(t, "none", Repository.Signing.SigningKey)
	})
}

func TestSettingRepositoryRootPath(t *testing.T) {
	t.Run("Absolute", func(t *testing.T) {
		require.NoError(t, loadRepositoryFromData(t, `
[repository]
ROOT = /path/to/git-repos
`))
		assert.Equal(t, "/path/to/git-repos", Repository.RootPath)
	})

	t.Run("Relative", func(t *testing.T) {
		require.NoError(t, loadRepositoryFromData(t, `
[repository]
ROOT = git-repos
`))
		assert.Equal(t, path.Join(AppWorkPath, "git-repos"), Repository.RootPath)
	})

	t.Run("Default", func(t *testing.T) {
		require.NoError(t, loadRepositoryFromData(t, ""))
		assert.Equal(t, path.Join(AppDataPath, "forgejo-repositories"), Repository.RootPath)
	})
}

func TestSSHInstanceKey(t *testing.T) {
	t.Run("Default value", func(t *testing.T) {
		require.Error(t, loadRepositoryFromData(t, `
[repository.signing]
FORMAT = ssh
`))
	})

	t.Run("No value", func(t *testing.T) {
		require.Error(t, loadRepositoryFromData(t, `
[repository.signing]
FORMAT = ssh
SIGNING_KEY =
`))
	})

	t.Run("None value", func(t *testing.T) {
		require.Error(t, loadRepositoryFromData(t, `
[repository.signing]
FORMAT = ssh
SIGNING_KEY = none
`))
	})

	t.Run("Key does not exist", func(t *testing.T) {
		require.Error(t, loadRepositoryFromData(t, `
[repository.signing]
FORMAT = ssh
SIGNING_KEY = /file/does/not/exist
`))
	})

	t.Run("Not a public SSH key", func(t *testing.T) {
		sshSigningKeyPath, err := filepath.Abs("../../tests/integration/ssh-signing-key")
		require.NoError(t, err)
		require.Error(t, loadRepositoryFromData(t, fmt.Sprintf(`
[repository.signing]
FORMAT = ssh
SIGNING_KEY = %s
`, sshSigningKeyPath)))
	})

	t.Run("Normal", func(t *testing.T) {
		sshSigningKeyPath, err := filepath.Abs("../../tests/integration/ssh-signing-key.pub")
		require.NoError(t, err)
		require.NoError(t, loadRepositoryFromData(t, fmt.Sprintf(`
[repository.signing]
FORMAT = ssh
SIGNING_KEY = %s
`, sshSigningKeyPath)))
		assert.NotNil(t, SSHInstanceKey)
		assert.Equal(t, "ssh-ed25519", SSHInstanceKey.Type())
		assert.EqualValues(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFeRC8GfFyXtiy0f1E7hLv77BXW7e68tFvIcs8/29YqH\n", ssh.MarshalAuthorizedKey(SSHInstanceKey))
	})
}
