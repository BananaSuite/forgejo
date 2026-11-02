// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-IDentifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"strings"

	"forgejo.org/modules/json"
	"forgejo.org/modules/packages/debian"

	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "update Debian package metadata in package_version.metadata.json",
		Upgrade:     updateDebianPackageMetadata,
	})
}

func updateDebianPackageMetadata(x *xorm.Engine) error {
	session := x.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return err
	}

	type PackageControl struct {
		VersionID int64  `xorm:"id"`
		Control   string `xorm:"value"`
	}
	var pcs []PackageControl
	if err := session.SQL(`SELECT package_version.id, package_property.value
FROM package, package_file, package_property, package_version
WHERE package.type == "debian"
AND NOT package.is_internal
AND package.id == package_version.package_id
AND package_version.id == package_file.version_id
AND package_file.id == package_property.ref_id
AND package_property.name = "debian.control"`).Find(&pcs); err != nil {
		return err
	}

	type PackageVersion struct {
		ID           int64
		MetadataJSON string `xorm:"metadata_json"`
	}
	for _, pc := range pcs {
		debianPackage, err := debian.ParseControlFile(strings.NewReader(pc.Control))
		if err != nil {
			return err
		}
		metadataJSON, err := json.Marshal(debianPackage.Metadata)
		if err != nil {
			return err
		}
		if _, err := session.Table("package_version").ID(pc.VersionID).Update(&PackageVersion{
			ID:           pc.VersionID,
			MetadataJSON: string(metadataJSON),
		}); err != nil {
			return err
		}
	}
	return session.Commit()
}
