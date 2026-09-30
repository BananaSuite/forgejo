// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
export function initPreviewServiceMessageButton() {
  const previewSMButton = document.querySelector('#preview-service-message-button');
  previewSMButton?.addEventListener('click', () => {
    const form: HTMLFormElement = document.querySelector('.service-message.form');
    if (!form.checkValidity()) {
      form.reportValidity();
      return false;
    }
    var action = form.action;
    var newAct = action.replace("/admin/service_message?sm_type=modal", "/admin/service_message/preview?sm_type=modal&status=show");
    form.action = newAct;
  });
}
