// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package debian

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"forgejo.org/modules/zstd"

	"github.com/blakesmith/ar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

const (
	packageName         = "gitea"
	packageVersion      = "0:1.0.1-te~st"
	version             = "1.0.1-te~st"
	packageArchitecture = "amd64"
	packageAuthor       = "KN4CK3R"
	packageDescription  = `single line synopsis
 extended description
 with multiple
 lines.
 .
 New paragraph!
 With verbatim list:
  * item a
  * item b
 extending b
 .
    indented with 4 spaces`
	description = `single line synopsis
extended description
with multiple
lines.

New paragraph!
With verbatim list:
 * item a
 * item b
extending b

   indented with 4 spaces`
	projectURL     = "https://gitea.io"
	packageDepends = "a (>= 1.0) | b, c"
)

func TestParsePackage(t *testing.T) {
	createArchive := func(files map[string][]byte) io.Reader {
		var buf bytes.Buffer
		aw := ar.NewWriter(&buf)
		aw.WriteGlobalHeader()
		for filename, content := range files {
			hdr := &ar.Header{
				Name: filename,
				Mode: 0o600,
				Size: int64(len(content)),
			}
			aw.WriteHeader(hdr)
			aw.Write(content)
		}
		return &buf
	}

	t.Run("MissingControlFile", func(t *testing.T) {
		data := createArchive(map[string][]byte{"dummy.txt": {}})

		p, err := ParsePackage(data)
		assert.Nil(t, p)
		require.ErrorIs(t, err, ErrMissingControlFile)
	})

	t.Run("Compression", func(t *testing.T) {
		t.Run("Unsupported", func(t *testing.T) {
			data := createArchive(map[string][]byte{"control.tar.foo": {}})

			p, err := ParsePackage(data)
			assert.Nil(t, p)
			require.ErrorIs(t, err, ErrUnsupportedCompression)
		})

		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		tw.WriteHeader(&tar.Header{
			Name: "control",
			Mode: 0o600,
			Size: 50,
		})
		tw.Write([]byte("Package: gitea\nVersion: 1.0.0\nArchitecture: amd64\n"))
		tw.Close()

		cases := []struct {
			Extension     string
			WriterFactory func(io.Writer) io.WriteCloser
		}{
			{
				Extension: "",
				WriterFactory: func(w io.Writer) io.WriteCloser {
					return nopCloser{w}
				},
			},
			{
				Extension: ".gz",
				WriterFactory: func(w io.Writer) io.WriteCloser {
					return gzip.NewWriter(w)
				},
			},
			{
				Extension: ".xz",
				WriterFactory: func(w io.Writer) io.WriteCloser {
					xw, _ := xz.NewWriter(w)
					return xw
				},
			},
			{
				Extension: ".zst",
				WriterFactory: func(w io.Writer) io.WriteCloser {
					zw, _ := zstd.NewWriter(w)
					return zw
				},
			},
		}

		for _, c := range cases {
			t.Run(c.Extension, func(t *testing.T) {
				var cbuf bytes.Buffer
				w := c.WriterFactory(&cbuf)
				w.Write(buf.Bytes())
				w.Close()

				data := createArchive(map[string][]byte{"control.tar" + c.Extension: cbuf.Bytes()})

				p, err := ParsePackage(data)
				assert.NotNil(t, p)
				require.NoError(t, err)
				assert.Equal(t, "gitea", p.Name)

				t.Run("TrailingSlash", func(t *testing.T) {
					data := createArchive(map[string][]byte{"control.tar" + c.Extension + "/": cbuf.Bytes()})

					p, err := ParsePackage(data)
					assert.NotNil(t, p)
					require.NoError(t, err)
					assert.Equal(t, "gitea", p.Name)
				})
			})
		}
	})
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error {
	return nil
}

func TestParseControlFile(t *testing.T) {
	buildContent := func(name, version, architecture, description string) *bytes.Buffer {
		var buf bytes.Buffer
		buf.WriteString("Package: " + name + "\nVersion: " + version + "\nArchitecture: " + architecture + "\nMaintainer: " + packageAuthor + " <kn4ck3r@gitea.io>\nHomepage: " + projectURL + "\nDepends: " + packageDepends + "\nDescription: " + description)
		return &buf
	}

	t.Run("InvalidName", func(t *testing.T) {
		for _, name := range []string{"", "-cd"} {
			p, err := ParseControlFile(buildContent(name, packageVersion, packageArchitecture, packageDescription))
			assert.Nil(t, p)
			require.ErrorIs(t, err, ErrInvalidName)
		}
	})

	t.Run("InvalidVersion", func(t *testing.T) {
		for _, version := range []string{"", "1-", ":1.0", "1_0"} {
			p, err := ParseControlFile(buildContent(packageName, version, packageArchitecture, packageDescription))
			assert.Nil(t, p)
			require.Error(t, err)
			require.ErrorContains(t, err, "control file parsing failed: ")
		}
	})

	t.Run("InvalidArchitecture", func(t *testing.T) {
		p, err := ParseControlFile(buildContent(packageName, packageVersion, "", packageDescription))
		assert.Nil(t, p)
		require.ErrorIs(t, err, ErrInvalidArchitecture)
	})

	t.Run("ValidVersionEpoch", func(t *testing.T) {
		for _, version := range []string{"0:1.2.3-test", "1:1.2.3-test", "9:1.2.3-test", "10:1.2.3-test", "37:1.2.3-test", "99:1.2.3-test"} {
			p, err := ParseControlFile(buildContent(packageName, version, packageArchitecture, packageDescription))
			require.NoError(t, err)
			assert.NotNil(t, p)
		}
	})

	t.Run("Valid", func(t *testing.T) {
		content := buildContent(packageName, packageVersion, packageArchitecture, packageDescription)
		full := content.String()

		p, err := ParseControlFile(content)
		require.NoError(t, err)
		assert.NotNil(t, p)

		assert.Equal(t, packageName, p.Name)
		assert.Equal(t, version, p.Version)
		assert.Equal(t, packageArchitecture, p.Architecture)
		assert.Equal(t, description, p.Metadata.Description)
		assert.Equal(t, projectURL, p.Metadata.ProjectURL)
		assert.Equal(t, packageAuthor, p.Metadata.Maintainer)
		assert.Len(t, p.Metadata.Depends, 3)
		assert.Equal(t, "a", p.Metadata.Depends[0].Name)
		assert.Equal(t, "(>= 1.0)", p.Metadata.Depends[0].VersionRequirement)
		assert.False(t, p.Metadata.Depends[0].IsAlternative)
		assert.Equal(t, "b", p.Metadata.Depends[1].Name)
		assert.Empty(t, p.Metadata.Depends[1].VersionRequirement)
		assert.True(t, p.Metadata.Depends[1].IsAlternative)
		assert.Equal(t, "c", p.Metadata.Depends[2].Name)
		assert.Empty(t, p.Metadata.Depends[2].VersionRequirement)
		assert.False(t, p.Metadata.Depends[2].IsAlternative)
		assert.Equal(t, full, p.Control)
	})
}

func TestDebianDescriptionSynopsisAndBody(t *testing.T) {
	for description, expected := range map[string]Description{
		"only synopsis": {
			Synopsis:     "only synopsis",
			BodyMarkdown: "",
		},
		"\nonly body": {
			Synopsis:     "",
			BodyMarkdown: "only body",
		},
		"synopsis\nbody first\nparagraph\n\nsecond paragraph": {
			Synopsis:     "synopsis",
			BodyMarkdown: "body first\nparagraph\n\nsecond paragraph",
		},
		"synopsis\n indented\n  more indented\nnormal paragraph": {
			Synopsis:     "synopsis",
			BodyMarkdown: "```\n indented\n  more indented\n```\n\nnormal paragraph",
		},
	} {
		assert.Equal(t, expected, (&Metadata{
			Description: description,
		}).DescriptionSynopsisAndBody())
	}
}
