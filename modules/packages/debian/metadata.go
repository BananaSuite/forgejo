// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package debian

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"net/mail"
	"regexp"
	"strings"

	"forgejo.org/modules/util"
	"forgejo.org/modules/validation"
	"forgejo.org/modules/zstd"

	"github.com/blakesmith/ar"
	"github.com/ulikunitz/xz"
	"pault.ag/go/debian/control"
	"pault.ag/go/debian/dependency"
)

const (
	PropertyDistribution               = "debian.distribution"
	PropertyComponent                  = "debian.component"
	PropertyArchitecture               = "debian.architecture"
	PropertyControl                    = "debian.control"
	PropertyRepositoryIncludeInRelease = "debian.repository.include_in_release"

	SettingKeyPrivate = "debian.key.private"
	SettingKeyPublic  = "debian.key.public"

	RepositoryPackage = "_debian"
	RepositoryVersion = "_repository"

	controlTar = "control.tar"
)

var (
	ErrMissingControlFile     = util.NewInvalidArgumentErrorf("control file is missing")
	ErrMissingBinaryIndex     = util.NewInvalidArgumentErrorf("binary index is missing in control file")
	ErrUnsupportedCompression = util.NewInvalidArgumentErrorf("unsupported compression algorithm")
	ErrInvalidName            = util.NewInvalidArgumentErrorf("package name is invalid")
	ErrInvalidArchitecture    = util.NewInvalidArgumentErrorf("package architecture is invalid")

	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#source
	namePattern = regexp.MustCompile(`\A[a-z0-9][a-z0-9+-.]+\z`)
)

type Package struct {
	Name         string
	Version      string
	Architecture string
	Control      string
	Metadata     *Metadata
}

type Metadata struct {
	Maintainer   string        `json:"maintainer,omitempty"`
	ProjectURL   string        `json:"project_url,omitempty"`
	Description  string        `json:"description,omitempty"`
	Dependencies []string      `json:"dependencies,omitempty"`
	Depends      []*Dependency `json:"depends,omitempty"`
	Suggests     []*Dependency `json:"suggests,omitempty"`
	Recommends   []*Dependency `json:"recommends,omitempty"`
	PreDepends   []*Dependency `json:"pre_depends,omitempty"`
	Breaks       []*Dependency `json:"breaks,omitempty"`
	Conflicts    []*Dependency `json:"conflicts,omitempty"`
	Replaces     []*Dependency `json:"replaces,omitempty"`
	Enhances     []*Dependency `json:"enhances,omitempty"`
	Provides     []*Dependency `json:"provides,omitempty"`
	BuiltUsing   []*Dependency `json:"built_using,omitempty"`
}

type Dependency struct {
	Name               string `json:"name"`
	VersionRequirement string `json:"version_requirement,omitempty"`
	IsAlternative      bool   `json:"is_alternative,omitempty"`
}

type Description struct {
	Synopsis     string
	BodyMarkdown string
}

// DescriptionSynopsisAndBody separates the synopsis from the body and marks up latter in Markdown format
func (m *Metadata) DescriptionSynopsisAndBody() Description {
	descriptionSplit := strings.Split(m.Description, "\n")
	synopsis := strings.TrimSpace(descriptionSplit[0])
	var bodyMarkdown strings.Builder
	inPre := false
	for _, line := range descriptionSplit[1:] {
		if len(line) != 0 && line[0] == ' ' {
			if !inPre {
				bodyMarkdown.WriteString("\n```\n")
				inPre = true
			}
		} else if inPre {
			bodyMarkdown.WriteString("```\n\n")
			inPre = false
		}
		bodyMarkdown.WriteString(line + "\n")
	}
	return Description{
		Synopsis:     synopsis,
		BodyMarkdown: strings.Trim(bodyMarkdown.String(), "\n"),
	}
}

// GetDepends returns the dependencies prepared for the output table
// from the new Metadata.Depends or the legacy Metadata.Dependencies field
func (m *Metadata) GetDepends() (dependencies []*Dependency) {
	// first, check if Metadata.Depends is set (for new packages)
	if m.Depends != nil {
		return m.Depends
	}
	// second, try go-debian package parsing for old packages
	dep, err := dependency.Parse(strings.Join(m.Dependencies, ","))
	if err == nil {
		return getDependencies(*dep)
	}
	// third, if it fails, parse manually for old packages
	for _, dependencyLine := range m.Dependencies {
		dependencyAlternatives := strings.Split(dependencyLine, "|")
		for i, dep := range dependencyAlternatives {
			depSplit := strings.SplitN(strings.TrimSpace(dep), " ", 2)
			versionRequirement := ""
			if len(depSplit) == 2 {
				versionRequirement = depSplit[1]
			}
			dependencies = append(dependencies, &Dependency{
				Name:               depSplit[0],
				VersionRequirement: versionRequirement,
				IsAlternative:      i != 0,
			})
		}
	}
	return dependencies
}

// ParsePackage parses the Debian package file
// https://manpages.debian.org/bullseye/dpkg-dev/deb.5.en.html
func ParsePackage(r io.Reader) (*Package, error) {
	arr := ar.NewReader(r)

	for {
		hd, err := arr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(hd.Name, controlTar) {
			var inner io.Reader
			// https://man7.org/linux/man-pages/man5/deb-split.5.html#FORMAT
			// The file names might contain a trailing slash (since dpkg 1.15.6).
			switch strings.TrimSuffix(hd.Name[len(controlTar):], "/") {
			case "":
				inner = arr
			case ".gz":
				gzr, err := gzip.NewReader(arr)
				if err != nil {
					return nil, err
				}
				defer gzr.Close()

				inner = gzr
			case ".xz":
				xzr, err := xz.NewReader(arr)
				if err != nil {
					return nil, err
				}

				inner = xzr
			case ".zst":
				zr, err := zstd.NewReader(arr)
				if err != nil {
					return nil, err
				}
				defer zr.Close()

				inner = zr
			default:
				return nil, ErrUnsupportedCompression
			}

			tr := tar.NewReader(inner)
			for {
				hd, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, err
				}

				if hd.Typeflag != tar.TypeReg {
					continue
				}

				if hd.FileInfo().Name() == "control" {
					return ParseControlFile(tr)
				}
			}
		}
	}

	return nil, ErrMissingControlFile
}

// ParseControlFile parses a Debian control file to retrieve the metadata
func ParseControlFile(r io.Reader) (*Package, error) {
	var controlBuilder strings.Builder
	binaryIndices, err := control.ParseBinaryIndex(bufio.NewReader(io.TeeReader(r, &controlBuilder)))
	if err != nil {
		return nil, err
	}

	if len(binaryIndices) == 0 {
		return nil, ErrMissingBinaryIndex
	}
	binaryIndex := binaryIndices[0]

	p := &Package{
		Name:         binaryIndex.Package,
		Version:      binaryIndex.Version.String(),
		Architecture: binaryIndex.Architecture.String(),
		Control:      strings.TrimSpace(controlBuilder.String()),
		Metadata: &Metadata{
			Maintainer:  binaryIndex.Maintainer,
			ProjectURL:  binaryIndex.Homepage,
			Description: strings.TrimSpace(binaryIndex.Description),
			Depends:     getDependencies(binaryIndex.GetDepends()),
			Suggests:    getDependencies(binaryIndex.GetSuggests()),
			Recommends:  getDependencies(binaryIndex.GetRecommends()),
			PreDepends:  getDependencies(binaryIndex.GetPreDepends()),
			Breaks:      getDependencies(binaryIndex.GetBreaks()),
			Conflicts:   getDependencies(binaryIndex.GetConflicts()),
			Replaces:    getDependencies(binaryIndex.GetReplaces()),
			Enhances:    getDependencies(binaryIndex.GetEnhances()),
			Provides:    getDependencies(binaryIndex.GetProvides()),
			BuiltUsing:  getDependencies(binaryIndex.GetBuiltUsing()),
		},
	}

	if !namePattern.MatchString(p.Name) {
		return nil, ErrInvalidName
	}
	if p.Architecture == "" {
		return nil, ErrInvalidArchitecture
	}
	address, err := mail.ParseAddress(p.Metadata.Maintainer)
	if err == nil && address.Name != "" {
		p.Metadata.Maintainer = address.Name
	}
	if !validation.IsValidURL(p.Metadata.ProjectURL) {
		p.Metadata.ProjectURL = ""
	}

	return p, nil
}

// getDependencies returns the dependencies prepared for the output table with name,
// version and whether it is an alternative dependency
func getDependencies(dep dependency.Dependency) (dependencies []*Dependency) {
	for _, relation := range dep.Relations {
		for i, possibility := range relation.Possibilities {
			versionRequirement := ""
			if possibility.Version != nil {
				versionRequirement = possibility.Version.String()
			}
			dependencies = append(dependencies, &Dependency{
				Name:               possibility.Name,
				VersionRequirement: versionRequirement,
				IsAlternative:      i != 0,
			})
		}
	}
	return dependencies
}
