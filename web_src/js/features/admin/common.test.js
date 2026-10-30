import {afterEach, expect, test, vi} from 'vitest';
import {initAdminCommon} from './common.js';

vi.mock('../common-global.js', () => ({checkAppUrl: vi.fn()}));

afterEach(() => {
  document.body.innerHTML = '';
});

test.each(['new', 'edit'])('OIDC logout is optional and only shown for OpenID Connect (%s source)', (mode) => {
  document.body.innerHTML = `
    <div class="page-content admin ${mode} authentication">
      <select id="auth_type"><option value="6" selected>OAuth2</option></select>
      <input id="auth_name" value="example">
      <span id="oauth2-callback-url"></span>
      <div class="oauth2">
        <select id="oauth2_provider">
          <option value="openidConnect" selected>OpenID Connect</option>
          <option value="github">GitHub</option>
        </select>
        <input id="oauth2_use_custom_url" type="checkbox">
        <div class="open_id_connect_auto_discovery_url required"><input></div>
        <div class="open_id_connect_logout optional field">
          <input id="open_id_connect_enable_logout" type="checkbox" checked>
        </div>
      </div>
    </div>`;
  initAdminCommon();
  const option = document.querySelector('.open_id_connect_logout');
  const checkbox = document.querySelector('#open_id_connect_enable_logout');
  const provider = document.querySelector('#oauth2_provider');
  expect(option.classList.contains('tw-hidden')).toBe(false);
  expect(checkbox.required).toBe(false);
  provider.value = 'github';
  provider.dispatchEvent(new Event('change'));
  expect(option.classList.contains('tw-hidden')).toBe(true);
  provider.value = 'openidConnect';
  provider.dispatchEvent(new Event('change'));
  expect(option.classList.contains('tw-hidden')).toBe(false);
  expect(checkbox.checked).toBe(true);
  expect(checkbox.required).toBe(false);
});
