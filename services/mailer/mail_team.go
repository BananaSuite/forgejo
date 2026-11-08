// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"bytes"
	"context"

	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/translation"
)

const (
	tplAddTeamMemberMail base.TplName = "org/add_team_member"
)

// MailAddTeamMember sends an email when a user is added to a team
func MailAddTeamMember(ctx context.Context, member *user_model.User, team *org_model.Team) error {
	if setting.MailService == nil {
		return nil
	}

	org, err := user_model.GetUserByID(ctx, team.OrgID)
	if err != nil {
		return err
	}

	locale := translation.NewLocale(member.Language)

	teamURL := team.Link(ctx)

	subject := locale.TrString("mail.add_team_member.subject", team.Name, org.DisplayName())
	mailMeta := map[string]any{
		"locale":       locale,
		"Organization": org,
		"Team":         team,
		"Subject":      subject,
		"TeamURL":      teamURL,
	}
	var mailBody bytes.Buffer
	if err := bodyTemplates.ExecuteTemplate(&mailBody, string(tplAddTeamMemberMail), mailMeta); err != nil {
		log.Error("ExecuteTemplate [%s]: %v", string(tplAddTeamMemberMail)+"/body", err)
		return err
	}

	msg := NewMessage(member.EmailTo(), subject, mailBody.String())
	msg.Info = subject

	SendAsync(msg)

	return nil
}
