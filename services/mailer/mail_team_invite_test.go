// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"testing"

	org_model "forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/translation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamNotificationMail_test(t *testing.T) {
	defer test.MockVariableValue(&setting.Admin.DisableRegularOrgCreation, false)()
	defer test.MockVariableValue(&setting.Admin.SendNotificationEmailOnNewUser, true)()
	require.NoError(t, unittest.PrepareTestDatabase())
	inviter := new(user_model.User)
	inviter.Name = "new_user"
	inviter.Language = "en_US"
	inviter.IsAdmin = false
	inviter.Email = "new_user@example.com"
	inviter.LastLoginUnix = 1693648327
	inviter.CreatedUnix = 1693648027
	inviter.AllowCreateOrganization = true
	require.NoError(t, user_model.CreateUser(t.Context(), inviter, &user_model.CreateUserOverwriteOptions{AllowCreateOrganization: optional.Some(true)}))
	org := new(org_model.Organization)
	org.Name = "new_org"
	org.IsActive = true
	require.NoError(t, org_model.CreateOrganization(t.Context(), org, inviter))
	o, _ := org_model.GetOrgByName(t.Context(), org.Name)
	team, _ := o.GetOwnerTeam(t.Context())
	t.Run("with invitee", func(t *testing.T) {
		invitedUser := new(user_model.User)
		invitedUser.Name = "invited_user"
		invitedUser.Language = "de-DE"
		invitedUser.IsAdmin = false
		invitedUser.Email = "invited_user@example.com"
		invitedUser.LastLoginUnix = 1693648327
		invitedUser.CreatedUnix = 1693648027
		require.NoError(t, user_model.CreateUser(t.Context(), invitedUser))
		invite, _ := org_model.CreateTeamInviteForUser(t.Context(), inviter, invitedUser, team)

		called := false
		defer MockMailSettings(func(msgs ...*Message) {
			assert.Len(t, msgs, 1, "Test provides only one user, so only one email must be sent")
			assert.Equal(t, invitedUser.Email, msgs[0].To, "checks if the recipient is invited user")
			locale := translation.NewLocale(invite.InvitedUser.Language)
			subject := locale.TrString("mail.team_invite.subject", inviter.DisplayName(), org.DisplayName())
			assert.Equal(t, msgs[0].Subject, subject)
			AssertTranslatedLocale(t, msgs[0].Subject, "mail.team_invite.subject")
			called = true
		})()

		MailTeamInvite(t.Context(), inviter, team, invite)

		assert.True(t, called)
	})

	t.Run("without invitee", func(t *testing.T) {
		invite, _ := org_model.CreateTeamInviteByEmail(t.Context(), inviter, team, "non-user@example.com")

		called := false
		defer MockMailSettings(func(msgs ...*Message) {
			assert.Len(t, msgs, 1, "Test provides only one user, so only one email must be sent")
			assert.Equal(t, "non-user@example.com", msgs[0].To, "checks if the recipient is invited user")
			locale := translation.NewLocale(inviter.Language)
			subject := locale.TrString("mail.team_invite.subject", inviter.DisplayName(), org.DisplayName())
			assert.Equal(t, msgs[0].Subject, subject)
			AssertTranslatedLocale(t, msgs[0].Subject, "mail.team_invite.subject")
			called = true
		})()

		MailTeamInvite(t.Context(), inviter, team, invite)

		assert.True(t, called)
	})
}
