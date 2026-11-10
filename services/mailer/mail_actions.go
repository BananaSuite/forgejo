// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package mailer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	actions_model "forgejo.org/models/actions"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/actions"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/translation"
	actions_service "forgejo.org/services/actions"
)

const (
	tplActionJobFailure         base.TplName = "actions/job_failure"
	actionJobFailureLogMaxLines              = 500
	actionJobFailureLogMaxBytes              = 128 * 1024
)

func readLastActionJobLogLines(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, actions.MaxStoredLineSize), actions.MaxStoredLineSize)

	var lines []string
	size := 0
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		size += len(line) + 1
		for len(lines) > actionJobFailureLogMaxLines || size > actionJobFailureLogMaxBytes {
			size -= len(lines[0]) + 1
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("could not read job log: %w", err)
	}
	return strings.Join(lines, "\n"), nil
}

var sendActionRunJobFailureNotification = func(ctx context.Context, job *actions_model.ActionRunJob) error {
	if !job.Status.IsFailure() {
		return nil
	}

	if setting.MailService == nil {
		// No mail service configured
		return nil
	}

	if !job.Run.NotifyEmail {
		return nil
	}

	user := job.Run.TriggerUser
	// this happens e.g. when this is a scheduled run
	if user.IsSystem() {
		user = job.Run.Repo.Owner
	}
	if user.IsSystem() || user.Email == "" {
		return nil
	}

	if user.EmailNotificationsPreference == user_model.EmailNotificationsDisabled {
		return nil
	}

	jobLink, err := job.HTMLURL(ctx)
	if err != nil {
		return fmt.Errorf("could not generate link to job results: %w", err)
	}

	var jobLog string
	reader, _, _, err := actions_service.OpenJobLogReader(
		ctx, job.Run.Repo, job.ID, optional.None[int64](), optional.None[int](),
	)
	if err != nil {
		log.Warn("SendActionRunJobFailureNotification: could not open log for job %d: %v", job.ID, err)
	} else {
		jobLog, err = readLastActionJobLogLines(reader)
		if err != nil {
			log.Warn("SendActionRunJobFailureNotification: could not read log for job %d: %v", job.ID, err)
			jobLog = ""
		}
		if err := reader.Close(); err != nil {
			log.Warn("SendActionRunJobFailureNotification: could not close log for job %d: %v", job.ID, err)
		}
	}

	locale := translation.NewLocale(user.Language)
	data := map[string]any{
		"locale":   locale,
		"Language": locale.Language(),
		"Link":     jobLink,
		"Job":      job,
		"Log":      jobLog,
	}

	var content bytes.Buffer
	if err := bodyTemplates.ExecuteTemplate(&content, string(tplActionJobFailure), data); err != nil {
		return err
	}

	subject := fmt.Sprintf("[%[1]s] %s", job.Run.Repo.FullName(),
		locale.TrString("mail.actions.job_failure_subject", job.Name))
	msg := NewMessage(user.EmailTo(), subject, content.String())
	msg.Info = subject
	SendAsync(msg)

	return nil
}
