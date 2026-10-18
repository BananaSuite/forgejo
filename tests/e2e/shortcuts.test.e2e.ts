// @watch start
// templates/shared/user/shortcuts_popup.tmpl
// web_src/js/features/user-shortcuts.ts
// @watch end

import {expect, type Response} from '@playwright/test';
import {test, login_user, login} from './utils_e2e.ts';

test.skip(({isMobile}) => isMobile, 'Desktop only');

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Code shortcuts', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);

  expect(
    (await page.goto('/user2/repo2/src/branch/master/test.xml'))?.status(),
  ).toBe(200);
  await page.keyboard.press('t');
  await expect(page).toHaveURL('/user2/repo2/find/branch/master');
  await expect(page.locator('#repo-file-find-input')).toBeFocused();

  expect((await page.goto('/user2/repo1/issues'))?.status()).toBe(200);
  await page.keyboard.press('s');
  await expect(page.locator('input[type="search"]').first()).toBeFocused();
});

test('Open dialog and persist enable setting', async ({
  browser,
}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/');
  await page.keyboard.press('Shift+?');
  const dialog = page.locator('#shortcuts');
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText('Find a file')).toBeHidden();
  await expect(dialog.getByText('Focus on searchbar')).toBeVisible();

  const checkbox = dialog.getByRole('checkbox');
  await expect(checkbox).toBeChecked();

  const checkResponse = async (value: boolean) => {
    expect(
      (
        (await (
          await page.waitForResponse(
            (resp: Response) =>
              resp.url().includes('/user/settings/appearance/shortcuts') &&
              resp.request().method() === 'POST' &&
              resp.status() === 200,
          )
        ).json()) as {enable_shortcuts: boolean}
      ).enable_shortcuts,
    ).toBe(value);
  };

  await Promise.all([checkResponse(false), checkbox.uncheck()]);
  await expect(checkbox).not.toBeChecked();
  await Promise.all([checkResponse(true), checkbox.check()]);

  await page.keyboard.press('ArrowDown');
  await expect(dialog.locator('input').first()).toHaveValue('repo');
  await expect(dialog.getByText('Find a file')).toBeVisible();
  await page.keyboard.press('ArrowUp');
  await expect(dialog.locator('input').first()).toHaveValue('global');
  await expect(dialog.getByText('Focus on searchbar')).toBeVisible();
  await page.keyboard.press('Escape');
  await dialog.isHidden();
});
