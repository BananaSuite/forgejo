import {POST} from '../modules/fetch.ts';

function initEnableCheckbox() {
  const checkbox = document.querySelector<HTMLInputElement>(
    '#shortcuts [type="checkbox"]',
  );
  checkbox?.addEventListener('change', () => {
    const enable = checkbox.checked;
    if (enable) document.addEventListener('keydown', onKeydown);
    else document.removeEventListener('keydown', onKeydown);
    const data = new URLSearchParams();
    if (enable) {
      data.set('enable_shortcuts', 'on');
    }
    POST(`${window.config.appSubUrl}/user/settings/appearance/shortcuts`, {
      headers: {Accept: 'application/json'},
      data,
    }).catch(console.error);
  });
}

function onKeydown(e: KeyboardEvent) {
  if (
    e.key === 'Escape' &&
    e.target instanceof HTMLInputElement &&
    e.target.type === 'search' &&
    e.target.value === ''
  ) {
    e.target.blur();
    return;
  }

  if (
    (e.altKey ||
      e.ctrlKey ||
      e.metaKey ||
      e.shiftKey ||
      !(e.target instanceof Element) ||
      e.target.closest('input, textarea, [contenteditable]')) &&
    (e.target as HTMLInputElement).type !== 'checkbox'
  ) return;

  switch (e.key) {
    case 's':
      document.querySelector<HTMLInputElement>('input[type="search"]')?.focus();
      e.preventDefault();
      break;
    case 't': {
      const findFileUrl = document.querySelector<HTMLElement>(
        '[data-find-file-url]',
      )?.dataset.findFileUrl;
      if (findFileUrl) window.location.assign(findFileUrl);
      break;
    }
  }
}

export function initUserShortcuts() {
  initEnableCheckbox();
  if (
    document.querySelector<HTMLInputElement>('#shortcuts [type="checkbox"]')
      ?.checked
  ) {
    document.addEventListener('keydown', onKeydown);
  }
  document.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === '?') {
      document.querySelector<HTMLDialogElement>('#shortcuts')?.showModal();
    }
  });
}
