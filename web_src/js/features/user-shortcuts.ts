import {POST} from '../modules/fetch.ts';

function initPopupTabs() {
  const dialog = document.querySelector<HTMLDialogElement>('#shortcuts');
  const inputs = dialog.querySelectorAll<HTMLInputElement>('[type="radio"]');
  const lists = dialog.querySelectorAll('ul');
  dialog.addEventListener('close', () => inputs[0].click());
  const fn = (value: string) => {
    for (const ls of lists) {
      const active = ls.dataset.tab !== value;
      ls.classList.toggle('tw-invisible', active);
      ls.toggleAttribute('aria-hidden', active);
    }
  };
  for (const input of inputs) {
    input.addEventListener('change', () => fn(input.value));
  }
  new MutationObserver(([mut], _) => {
    const el = mut.target as HTMLElement;
    if (el.getAttribute('aria-expanded') === 'true') return;
    const value = el.querySelector<HTMLInputElement>('.selected input')?.value;
    if (value) fn(value);
  }).observe(dialog.querySelector('.dropdown'), {
    attributeFilter: ['aria-expanded'],
  });
}

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
    case 'ArrowUp': {
      const dialog = document.querySelector<HTMLDialogElement>('#shortcuts');
      if (!dialog.open || dialog.querySelector('.visible')) return;
      (
        document.querySelector('#shortcuts .active')
          ?.previousElementSibling as HTMLElement
      )?.click();
      break;
    }
    case 'ArrowDown': {
      const dialog = document.querySelector<HTMLDialogElement>('#shortcuts');
      if (!dialog.open || dialog.querySelector('.visible')) return;
      (
        document.querySelector('#shortcuts .active')
          ?.nextElementSibling as HTMLElement
      )?.click();
      break;
    }
  }
}

export function initUserShortcuts() {
  initPopupTabs();
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
