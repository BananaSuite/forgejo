// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"testing"

	org_model "forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/translation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTeamMemberMail(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	team := unittest.AssertExistsAndLoadBean(t, &org_model.Team{ID: 1})
	org := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: team.OrgID})

	called := false
	defer MockMailSettings(func(msgs ...*Message) {
		assert.Len(t, msgs, 1, "only one email must be sent")
		assert.Equal(t, user.EmailTo(), msgs[0].To, "checks if the recipient is the user added to the team")
		locale := translation.NewLocale("en-US")
		subject := locale.TrString("mail.add_team_member.subject", team.Name, org.DisplayName())
		assert.Equal(t, msgs[0].Subject, subject)
		AssertTranslatedLocale(t, msgs[0].Subject, "mail.add_team_member.subject")
		called = true
	})()

	MailAddTeamMember(t.Context(), user, team)

	assert.True(t, called)
}
