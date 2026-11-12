// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"forgejo.org/modules/log"

	"golang.org/x/crypto/ssh"
)

var (
	ScriptType                    string
	Repository                    RepositorySetting
	MaxUserCardsPerPage           int
	MaxForksPerPage               int
	SSHInstanceKey                ssh.PublicKey
	downloadSourceArchiveDisabled map[bool]map[string]map[bool]map[string]bool
)

var (
	disableDownloadSourceArchiveReferences = []string{"default-branch", "non-default-branch", "commit", "tag"}
	disableDownloadSourceArchiveExtensions = []string{".zip", ".tar.gz", ".bundle"}
	disableDownloadSourceArchivePattern    = regexp.MustCompile(`^(?P<reference>any-branch|default-branch|non-default-branch|commit|tag|\*|)(?P<subdir>:?)(?P<extension>\.zip|\.tar\.gz|\.bundle|\.\*|\.)?$`)
)

// enumerates all the policy repository creating
const (
	RepoCreatingLastUserVisibility = "last"
	RepoCreatingPrivate            = "private"
	RepoCreatingPublic             = "public"
)

// Repository settings
type RepositorySetting struct {
	RootPath                                     string `ini:"ROOT"`
	DetectedCharsetsOrder                        []string
	DetectedCharsetScore                         map[string]int `ini:"-"`
	AnsiCharset                                  string
	ForcePrivate                                 bool
	DefaultPrivate                               string
	DefaultPushCreatePrivate                     bool
	MaxCreationLimit                             int
	PreferredLicenses                            []string
	DisableHTTPGit                               bool `ini:"DISABLE_HTTP_GIT"`
	AccessControlAllowOrigin                     string
	UseCompatSSHURI                              bool   `ini:"USE_COMPAT_SSH_URI"`
	GoGetCloneURLProtocol                        string `ini:"GO_GET_CLONE_URL_PROTOCOL"`
	DefaultCloseIssuesViaCommitsInAnyBranch      bool
	EnablePushCreateUser                         bool
	EnablePushCreateOrg                          bool
	DisabledRepoUnits                            []string
	DefaultRepoUnits                             []string
	DefaultForkRepoUnits                         []string
	DefaultMirrorRepoUnits                       []string
	PrefixArchiveFiles                           bool
	DisableMigrations                            bool
	DisableStars                                 bool
	DisableForks                                 bool
	DefaultBranch                                string
	AllowAdoptionOfUnadoptedRepositories         bool
	AllowDeleteOfUnadoptedRepositories           bool
	DisableDownloadSourceArchives                []string
	DisableDownloadSourceArchivesUnauthenticated []string
	AllowForkWithoutMaximumLimit                 bool

	// Repository editor settings
	Editor struct {
		LineWrapExtensions []string
	} `ini:"repository.editor"`

	// Repository upload settings
	Upload struct {
		Enabled      bool
		TempPath     string
		AllowedTypes string
		FileMaxSize  int64
		MaxFiles     int
	} `ini:"repository.upload"`

	// Repository local settings
	Local struct {
		LocalCopyPath string
	} `ini:"repository.local"`

	// Pull request settings
	PullRequest struct {
		WorkInProgressPrefixes                   []string
		CloseKeywords                            []string
		ReopenKeywords                           []string
		DefaultMergeStyle                        string
		DefaultMergeMessageCommitsLimit          int
		DefaultMergeMessageSize                  int
		DefaultMergeMessageAllAuthors            bool
		DefaultMergeMessageMaxApprovers          int
		DefaultMergeMessageOfficialApproversOnly bool
		DefaultUpdateStyle                       string
		PopulateSquashCommentWithCommitMessages  bool
		RetargetChildrenOnMerge                  bool
		KeepForcePushRefs                        bool
	} `ini:"repository.pull-request"`

	// Issue Setting
	Issue struct {
		LockReasons []string
		MaxPinned   int
	} `ini:"repository.issue"`

	Release struct {
		AllowedTypes     string
		DefaultPagingNum int
	} `ini:"repository.release"`

	Signing struct {
		SigningKey        string
		SigningName       string
		SigningEmail      string
		Format            string
		InitialCommit     []string
		CRUDActions       []string `ini:"CRUD_ACTIONS"`
		Merges            []string
		Wiki              []string
		DefaultTrustModel string
	} `ini:"repository.signing"`

	SettableFlags []string
	EnableFlags   bool
}

func loadRepositoryDefaults() {
	ScriptType = "bash"
	Repository = RepositorySetting{
		RootPath: path.Join(AppDataPath, "forgejo-repositories"),
		DetectedCharsetsOrder: []string{
			"UTF-8",
			"UTF-16BE",
			"UTF-16LE",
			"UTF-32BE",
			"UTF-32LE",
			"ISO-8859-1",
			"windows-1252",
			"ISO-8859-2",
			"windows-1250",
			"ISO-8859-5",
			"ISO-8859-6",
			"ISO-8859-7",
			"windows-1253",
			"ISO-8859-8-I",
			"windows-1255",
			"ISO-8859-8",
			"windows-1251",
			"windows-1256",
			"KOI8-R",
			"ISO-8859-9",
			"windows-1254",
			"Shift_JIS",
			"GB18030",
			"EUC-JP",
			"EUC-KR",
			"Big5",
			"ISO-2022-JP",
			"ISO-2022-KR",
			"ISO-2022-CN",
			"IBM424_rtl",
			"IBM424_ltr",
			"IBM420_rtl",
			"IBM420_ltr",
		},
		DetectedCharsetScore:                         map[string]int{},
		AnsiCharset:                                  "",
		ForcePrivate:                                 false,
		DefaultPrivate:                               RepoCreatingLastUserVisibility,
		DefaultPushCreatePrivate:                     true,
		MaxCreationLimit:                             -1,
		PreferredLicenses:                            []string{"Apache-2.0", "MIT"},
		DisableHTTPGit:                               false,
		AccessControlAllowOrigin:                     "",
		UseCompatSSHURI:                              true,
		GoGetCloneURLProtocol:                        "https",
		DefaultCloseIssuesViaCommitsInAnyBranch:      false,
		EnablePushCreateUser:                         false,
		EnablePushCreateOrg:                          false,
		DisabledRepoUnits:                            []string{},
		DefaultRepoUnits:                             []string{},
		DefaultForkRepoUnits:                         []string{},
		DefaultMirrorRepoUnits:                       []string{},
		PrefixArchiveFiles:                           true,
		DisableMigrations:                            false,
		DisableStars:                                 false,
		DisableForks:                                 false,
		DefaultBranch:                                "main",
		AllowAdoptionOfUnadoptedRepositories:         false,
		AllowDeleteOfUnadoptedRepositories:           false,
		DisableDownloadSourceArchives:                []string{},
		DisableDownloadSourceArchivesUnauthenticated: []string{},
		AllowForkWithoutMaximumLimit:                 true,

		// Repository editor settings
		Editor: struct {
			LineWrapExtensions []string
		}{
			LineWrapExtensions: strings.Split(".txt,.md,.markdown,.mdown,.mkd,.livemd,", ","),
		},

		// Repository upload settings
		Upload: struct {
			Enabled      bool
			TempPath     string
			AllowedTypes string
			FileMaxSize  int64
			MaxFiles     int
		}{
			Enabled:      true,
			TempPath:     path.Join(AppWorkPath, "data/tmp/uploads"),
			AllowedTypes: "",
			FileMaxSize:  50,
			MaxFiles:     5,
		},

		// Repository local settings
		Local: struct {
			LocalCopyPath string
		}{
			LocalCopyPath: "tmp/local-repo",
		},

		// Pull request settings
		PullRequest: struct {
			WorkInProgressPrefixes                   []string
			CloseKeywords                            []string
			ReopenKeywords                           []string
			DefaultMergeStyle                        string
			DefaultMergeMessageCommitsLimit          int
			DefaultMergeMessageSize                  int
			DefaultMergeMessageAllAuthors            bool
			DefaultMergeMessageMaxApprovers          int
			DefaultMergeMessageOfficialApproversOnly bool
			DefaultUpdateStyle                       string
			PopulateSquashCommentWithCommitMessages  bool
			RetargetChildrenOnMerge                  bool
			KeepForcePushRefs                        bool
		}{
			WorkInProgressPrefixes: []string{"WIP:", "[WIP]"},
			// Same as GitHub. See
			// https://help.github.com/articles/closing-issues-via-commit-messages
			CloseKeywords:                            strings.Split("close,closes,closed,fix,fixes,fixed,resolve,resolves,resolved", ","),
			ReopenKeywords:                           strings.Split("reopen,reopens,reopened", ","),
			DefaultMergeStyle:                        "merge",
			DefaultMergeMessageCommitsLimit:          50,
			DefaultMergeMessageSize:                  5 * 1024,
			DefaultMergeMessageAllAuthors:            false,
			DefaultMergeMessageMaxApprovers:          10,
			DefaultMergeMessageOfficialApproversOnly: true,
			DefaultUpdateStyle:                       "merge",
			PopulateSquashCommentWithCommitMessages:  false,
			RetargetChildrenOnMerge:                  true,
			KeepForcePushRefs:                        true,
		},

		// Issue settings
		Issue: struct {
			LockReasons []string
			MaxPinned   int
		}{
			LockReasons: strings.Split("Too heated,Off-topic,Spam,Resolved", ","),
			MaxPinned:   3,
		},

		Release: struct {
			AllowedTypes     string
			DefaultPagingNum int
		}{
			AllowedTypes:     "",
			DefaultPagingNum: 10,
		},

		// Signing settings
		Signing: struct {
			SigningKey        string
			SigningName       string
			SigningEmail      string
			Format            string
			InitialCommit     []string
			CRUDActions       []string `ini:"CRUD_ACTIONS"`
			Merges            []string
			Wiki              []string
			DefaultTrustModel string
		}{
			SigningKey:        "default",
			SigningName:       "",
			SigningEmail:      "",
			Format:            "openpgp",
			InitialCommit:     []string{"always"},
			CRUDActions:       []string{"pubkey", "twofa", "parentsigned"},
			Merges:            []string{"pubkey", "twofa", "basesigned", "commitssigned"},
			Wiki:              []string{"never"},
			DefaultTrustModel: "collaborator",
		},

		EnableFlags: false,
	}
	// MaxUserCardsPerPage sets maximum amount of watchers and stargazers shown per page
	// those pages use 2 or 3 column layout, so the value should be divisible by 2 and 3
	MaxUserCardsPerPage = 36
	// MaxForksPerPage sets maximum amount of forks shown per page
	MaxForksPerPage = 40
}

func loadRepositoryFrom(rootCfg ConfigProvider) error {
	loadRepositoryDefaults()
	// Determine and create root git repository path.
	sec := rootCfg.Section("repository")
	defaultDetectedCharsetsOrder := make([]string, 0, len(Repository.DetectedCharsetsOrder))
	for _, charset := range Repository.DetectedCharsetsOrder {
		defaultDetectedCharsetsOrder = append(defaultDetectedCharsetsOrder, strings.ToLower(strings.TrimSpace(charset)))
	}
	ScriptType = sec.Key("SCRIPT_TYPE").MustString("bash")

	if _, err := exec.LookPath(ScriptType); err != nil {
		log.Warn("SCRIPT_TYPE %q is not on the current PATH. Are you sure that this is the correct SCRIPT_TYPE?", ScriptType)
	}

	if err := sec.MapTo(&Repository); err != nil {
		return fmt.Errorf("Failed to map Repository settings: %v", err)
	}

	if !filepath.IsAbs(Repository.RootPath) {
		Repository.RootPath = filepath.Join(AppWorkPath, Repository.RootPath)
	} else {
		Repository.RootPath = filepath.Clean(Repository.RootPath)
	}

	if !rootCfg.Section("packages").Key("ENABLED").MustBool(Packages.Enabled) {
		Repository.DisabledRepoUnits = append(Repository.DisabledRepoUnits, "repo.packages")
	}

	if !rootCfg.Section("actions").Key("ENABLED").MustBool(Actions.Enabled) {
		Repository.DisabledRepoUnits = append(Repository.DisabledRepoUnits, "repo.actions")
	}

	// Handle default trustmodel settings
	Repository.Signing.DefaultTrustModel = strings.ToLower(strings.TrimSpace(Repository.Signing.DefaultTrustModel))
	if Repository.Signing.DefaultTrustModel == "default" {
		Repository.Signing.DefaultTrustModel = "collaborator"
	}

	// Handle preferred charset orders
	preferred := make([]string, 0, len(Repository.DetectedCharsetsOrder))
	for _, charset := range Repository.DetectedCharsetsOrder {
		canonicalCharset := strings.ToLower(strings.TrimSpace(charset))
		preferred = append(preferred, canonicalCharset)
		// remove it from the defaults
		for i, charset := range defaultDetectedCharsetsOrder {
			if charset == canonicalCharset {
				defaultDetectedCharsetsOrder = append(defaultDetectedCharsetsOrder[:i], defaultDetectedCharsetsOrder[i+1:]...)
				break
			}
		}
	}

	i := 0
	for _, charset := range preferred {
		// Add the defaults
		if charset == "defaults" {
			for _, charset := range defaultDetectedCharsetsOrder {
				canonicalCharset := strings.ToLower(strings.TrimSpace(charset))
				if _, has := Repository.DetectedCharsetScore[canonicalCharset]; !has {
					Repository.DetectedCharsetScore[canonicalCharset] = i
					i++
				}
			}
			continue
		}
		if _, has := Repository.DetectedCharsetScore[charset]; !has {
			Repository.DetectedCharsetScore[charset] = i
			i++
		}
	}

	if !filepath.IsAbs(Repository.Upload.TempPath) {
		Repository.Upload.TempPath = path.Join(AppWorkPath, Repository.Upload.TempPath)
	}

	if err := loadRepoArchiveFrom(rootCfg); err != nil {
		return fmt.Errorf("loadRepoArchiveFrom: %v", err)
	}
	Repository.EnableFlags = sec.Key("ENABLE_FLAGS").MustBool()

	if Repository.Signing.Format == "ssh" {
		if Repository.Signing.SigningKey == "none" || Repository.Signing.SigningKey == "" || Repository.Signing.SigningKey == "default" {
			return fmt.Errorf("SIGNING_KEY key must be specified, if signing format ssh is chosen")
		}
		sshPublicKey, err := os.ReadFile(Repository.Signing.SigningKey)
		if err != nil {
			return fmt.Errorf("Could not read repository signing key in %q: %v", Repository.Signing.SigningKey, err)
		}
		SSHInstanceKey, _, _, _, err = ssh.ParseAuthorizedKey(sshPublicKey)
		if err != nil {
			return fmt.Errorf("Could not parse the SSH signing key %q: %v", sshPublicKey, err)
		}
	}

	// Prepare and fill map for IsRepositoryDownloadSourceArchiveDisabled function
	downloadSourceArchiveDisabled = make(map[bool]map[string]map[bool]map[string]bool)
	for _, isSigned := range []bool{true, false} {
		downloadSourceArchiveDisabled[isSigned] = make(map[string]map[bool]map[string]bool)
		for _, reference := range disableDownloadSourceArchiveReferences {
			downloadSourceArchiveDisabled[isSigned][reference] = make(map[bool]map[string]bool)
			for _, isSubdir := range []bool{true, false} {
				downloadSourceArchiveDisabled[isSigned][reference][isSubdir] = make(map[string]bool)
				for _, extension := range disableDownloadSourceArchiveExtensions {
					downloadSourceArchiveDisabled[isSigned][reference][isSubdir][extension] = false
				}
			}
		}
	}
	for _, value := range Repository.DisableDownloadSourceArchives {
		references, isSubdir, extensions, err := parseDisableDownloadSourceArchive(value)
		if err != nil {
			return fmt.Errorf("[repository.download].DISABLED_SOURCE_ARCHIVES` not well formatted: %v", err)
		}
		for _, reference := range references {
			for _, extension := range extensions {
				downloadSourceArchiveDisabled[true][reference][true][extension] = true
				downloadSourceArchiveDisabled[false][reference][true][extension] = true
				if !isSubdir {
					downloadSourceArchiveDisabled[true][reference][false][extension] = true
					downloadSourceArchiveDisabled[false][reference][false][extension] = true
				}
			}
		}
	}
	for _, value := range Repository.DisableDownloadSourceArchivesUnauthenticated {
		references, isSubdir, extensions, err := parseDisableDownloadSourceArchive(value)
		if err != nil {
			return fmt.Errorf("[repository.download].DISABLED_SOURCE_ARCHIVES_UNAUTHENTICATED` not well formatted: %v", err)
		}
		for _, reference := range references {
			for _, extension := range extensions {
				downloadSourceArchiveDisabled[false][reference][true][extension] = true
				if !isSubdir {
					downloadSourceArchiveDisabled[false][reference][false][extension] = true
				}
			}
		}
	}

	return nil
}

func parseDisableDownloadSourceArchive(value string) (references []string, isSubdir bool, extensions []string, err error) {
	switch value {
	case "":
		return references, isSubdir, extensions, errors.New("value is empty")
	case "true":
		return disableDownloadSourceArchiveReferences, false, disableDownloadSourceArchiveExtensions, nil
	case "false":
		return []string{}, false, []string{}, nil
	}
	submatches := disableDownloadSourceArchivePattern.FindStringSubmatch(value)
	if len(submatches) == 0 {
		return references, isSubdir, extensions, fmt.Errorf("value `%s` not valid", value)
	}
	switch submatches[1] {
	case "*", "":
		references = disableDownloadSourceArchiveReferences
	case "any-branch":
		references = []string{"default-branch", "non-default-branch"}
	default:
		if !slices.Contains(disableDownloadSourceArchiveReferences, submatches[1]) {
			// This should never be reached, because it is excluded from the regexp pattern
			return references, isSubdir, extensions, fmt.Errorf("reference `%s` not valid", submatches[1])
		}
		references = []string{submatches[1]}
	}
	isSubdir = len(submatches[2]) != 0
	switch submatches[3] {
	case ".*", ".", "":
		extensions = disableDownloadSourceArchiveExtensions
	default:
		if !slices.Contains(disableDownloadSourceArchiveExtensions, submatches[3]) {
			// This should never be reached, because it is excluded from the regexp pattern
			return references, isSubdir, extensions, fmt.Errorf("extension `%s` not valid", submatches[3])
		}
		extensions = []string{submatches[3]}
	}
	return references, isSubdir, extensions, nil
}

func IsRepositoryDownloadSourceArchiveDisabled(isSigned bool, reference string, isSubdir bool, extension string) (disabled bool, err error) {
	disabled, ok := downloadSourceArchiveDisabled[isSigned][reference][isSubdir][extension]
	if !ok {
		if _, ok := downloadSourceArchiveDisabled[isSigned]; !ok {
			err = fmt.Errorf("key error: isSigned: %v not valid", isSigned)
		} else if _, ok = downloadSourceArchiveDisabled[isSigned][reference]; !ok {
			err = fmt.Errorf("key error: reference: \"%s\" not valid", reference)
		} else if _, ok = downloadSourceArchiveDisabled[isSigned][reference][isSubdir]; !ok {
			err = fmt.Errorf("key error: isSubdir: %v not valid", isSubdir)
		} else if _, ok = downloadSourceArchiveDisabled[isSigned][reference][isSubdir][extension]; !ok {
			err = fmt.Errorf("key error: extension: \"%s\" not valid", extension)
		}
	}
	return disabled, err
}
