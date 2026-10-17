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

	invitedUser := new(user_model.User)
	invitedUser.Name = "invited_user"
	invitedUser.Language = "de-DE"
	invitedUser.IsAdmin = false
	invitedUser.Email = "invited_user@example.com"
	invitedUser.LastLoginUnix = 1693648327
	invitedUser.CreatedUnix = 1693648027
	require.NoError(t, user_model.CreateUser(t.Context(), invitedUser))
	invite, _ := org_model.CreateTeamInviteForUser(t.Context(), inviter, invitedUser, team)

	defer test.MockVariableValue(&setting.Admin.SendNotificationEmailOnNewUser, true)()

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
}
