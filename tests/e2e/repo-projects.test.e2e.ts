// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/projects/view.tmpl
// web_src/js/features/repo-projects.ts
// @watch end

import {expect, type Page} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({user: 'user2'});

test.describe('project columns', () => {
  test.beforeEach(async ({page}, testInfo) => {
    const resp = await page.goto('/user2/repo2/projects');
    expect(resp.status()).toBe(200);

    // create a project for this test (project IDs are not always predictable, but test name works ok)
    const testPath = testInfo.titlePath.join(' › ');
    const testTitle = `${testPath} (${testInfo.project.name})`; // e.g. "repo-projects › project columns › create (Mobile Chrome)"
    const projectName = `Test project: ${testTitle}`;
    await page.getByRole('link', {name: 'New project'}).click();
    await page.getByRole('textbox', {name: 'Title'}).fill(projectName);
    await page.getByRole('button', {name: 'Create project'}).click();
    await page.getByRole('link', {name: projectName}).click();
  });

  async function tryCreateColumn(page: Page, expectModalStillVisible: boolean = false) {
    const createModal = page.locator('dialog#new-project-column-item');
    await expect(createModal).toBeHidden();
    await page.getByRole('button', {name: 'New column'}).click();
    await expect(createModal).toBeVisible();
    await createModal.getByRole('textbox', {name: 'Name'}).fill('Test column');
    await createModal.getByRole('button', {name: 'Create column'}).click();
    if (expectModalStillVisible) {
      await expect(createModal).toBeVisible();
    } else {
      await expect(createModal).toBeHidden();
    }
  }

  test('create success', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    await expect(page.locator('.project-column', {hasText: 'Test column'})).toBeVisible();
    await expect(page.locator('.toast-body')).toBeHidden();
  });

  test('cancel create closes modal', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    const createModal = page.locator('dialog#new-project-column-item');
    await expect(createModal).toBeHidden();
    await page.getByRole('button', {name: 'New column'}).click();
    await expect(createModal).toBeVisible();
    await createModal.getByRole('textbox', {name: 'Name'}).fill('Test column');
    await createModal.getByRole('button', {name: 'Cancel'}).click();

    // create did not occur
    await expect(createModal).toBeHidden();
    await expect(page.locator('.project-column')).toBeHidden();
  });

  test('create failure displays toast', async ({page}) => {
    const pathname = await page.evaluate(() => window.location.pathname);
    await page.route(pathname, (route) => route.abort('internetdisconnected'));

    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page, true); // modal didn't go away
    await expect(page.locator('.toast-body', {hasText: 'Network error'})).toBeVisible();
  });

  test('edit success', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    const testColumn = page.locator('.project-column').first();
    const testColumnTitle = testColumn.locator('span.project-column-title-label'); // TODO: heading level 3 or 4
    await expect(testColumn).toBeVisible();
    await expect(testColumnTitle).toHaveText('Test column');
    await expect(testColumnTitle).not.toHaveText('Different test column');

    const editModal = page.locator('dialog[id^="edit-project-column-modal-"]');
    await expect(editModal).toBeHidden();
    await testColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await testColumn.getByRole('option', {name: 'Edit column'}).click();
    await expect(editModal).toBeVisible();
    await editModal.getByRole('textbox', {name: 'Name'}).fill('Different test column');
    await editModal.getByRole('button', {name: 'Edit column'}).click();

    // edit worked
    await expect(editModal).toBeHidden();
    await expect(testColumnTitle).not.toHaveText('Test column');
    await expect(testColumnTitle).toHaveText('Different test column');
    await expect(page.locator('.toast-body')).toBeHidden();
  });

  test('cancel edit closes modal', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    const testColumn = page.locator('.project-column').first();
    const testColumnTitle = testColumn.locator('span.project-column-title-label'); // TODO: heading level 3 or 4
    await expect(testColumn).toBeVisible();
    await expect(testColumnTitle).toHaveText('Test column');
    await expect(testColumnTitle).not.toHaveText('Different test column');

    const editModal = page.locator('dialog[id^="edit-project-column-modal-"]');
    await expect(editModal).toBeHidden();
    await testColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await testColumn.getByRole('option', {name: 'Edit column'}).click();
    await expect(editModal).toBeVisible();
    await editModal.getByRole('textbox', {name: 'Name'}).fill('Different test column');
    await editModal.getByRole('button', {name: 'Cancel'}).click();

    // edit did not occur
    await expect(editModal).toBeHidden();
    await expect(testColumnTitle).toHaveText('Test column');
    await expect(testColumnTitle).not.toHaveText('Different test column');
  });

  test('edit failure displays toast', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    const testColumn = page.locator('.project-column').first();
    const testColumnTitle = testColumn.locator('span.project-column-title-label'); // TODO: heading level 3 or 4
    await expect(testColumn).toBeVisible();
    await expect(testColumnTitle).toHaveText('Test column');
    await expect(testColumnTitle).not.toHaveText('Different test column');

    const pathname = await page.evaluate(() => window.location.pathname);
    await page.route(`${pathname}/*`, (route) => route.abort('internetdisconnected'));

    const editModal = page.locator('dialog[id^="edit-project-column-modal-"]');
    await expect(editModal).toBeHidden();
    await testColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await testColumn.getByRole('option', {name: 'Edit column'}).click();
    await expect(editModal).toBeVisible();
    await editModal.getByRole('textbox', {name: 'Name'}).fill('Different test column');
    await editModal.getByRole('button', {name: 'Edit column'}).click();

    // edit failed
    await expect(editModal).toBeVisible(); // modal didn't go away
    await expect(testColumnTitle).toHaveText('Test column'); // title never changed
    await expect(testColumnTitle).not.toHaveText('Different test column');
    await expect(page.locator('.toast-body', {hasText: 'Network error'})).toBeVisible();
  });

  test('delete success', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    await tryCreateColumn(page);
    await expect(page.locator('.project-column')).toHaveCount(2);
    const extraColumn = page.locator('.project-column').last();
    await extraColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await extraColumn.getByRole('option', {name: 'Delete column'}).click();
    const deleteModal = page.locator('dialog[id^="delete-project-column-modal-"]:modal');
    await expect(deleteModal).toBeVisible();
    await deleteModal.getByRole('button', {name: 'Confirm'}).click();

    // delete worked
    await expect(deleteModal).toBeHidden();
    await expect(page.locator('.project-column')).toHaveCount(1);
    await expect(page.locator('.toast-body')).toBeHidden();
    // TODO: test that a specific column was deleted, not just any column
  });

  test('cancel delete closes modal', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    await tryCreateColumn(page);
    await expect(page.locator('.project-column')).toHaveCount(2);
    const extraColumn = page.locator('.project-column').last();
    await extraColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await extraColumn.getByRole('option', {name: 'Delete column'}).click();
    const deleteModal = page.locator('dialog[id^="delete-project-column-modal-"]:modal');
    await expect(deleteModal).toBeVisible();
    await deleteModal.getByRole('button', {name: 'Cancel'}).click();

    // delete did not occur
    await expect(deleteModal).toBeHidden();
    await expect(page.locator('.project-column')).toHaveCount(2);
  });

  test('delete failure displays toast', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    await tryCreateColumn(page);
    await expect(page.locator('.project-column')).toHaveCount(2);
    const extraColumn = page.locator('.project-column').last();

    const pathname = await page.evaluate(() => window.location.pathname);
    await page.route(`${pathname}/*`, (route) => route.abort('internetdisconnected'));

    await extraColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await extraColumn.getByRole('option', {name: 'Delete column'}).click();
    const deleteModal = page.locator('dialog[id^="delete-project-column-modal-"]:modal');
    await expect(deleteModal).toBeVisible();
    await deleteModal.getByRole('button', {name: 'Confirm'}).click();

    // delete failed
    await expect(deleteModal).toBeVisible();
    await expect(page.locator('.project-column')).toHaveCount(2);
    await expect(page.locator('.toast-body', {hasText: 'Network error'})).toBeVisible();
  });

  test('set default success', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    await tryCreateColumn(page);
    await expect(page.locator('.project-column')).toHaveCount(2);
    const testColumn = page.locator('.project-column').last();

    const makeDefaultModal = page.locator('dialog[id^="default-project-column-modal-"]').last();
    await expect(makeDefaultModal).toBeHidden();
    await testColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await testColumn.getByRole('option', {name: 'Set default'}).click();
    await expect(makeDefaultModal).toBeVisible();
    await makeDefaultModal.getByRole('button', {name: 'Confirm'}).click();

    // default worked
    await expect(makeDefaultModal).toBeHidden();
    await expect(page.locator('.toast-body')).toBeHidden();
    // TODO: test that a specific column was defaulted
  });

  test('cancel set default closes modal', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    await tryCreateColumn(page);
    await expect(page.locator('.project-column')).toHaveCount(2);
    const testColumn = page.locator('.project-column').last();

    const makeDefaultModal = page.locator('dialog[id^="default-project-column-modal-"]').last();
    await expect(makeDefaultModal).toBeHidden();
    await testColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await testColumn.getByRole('option', {name: 'Set default'}).click();
    await expect(makeDefaultModal).toBeVisible();
    await makeDefaultModal.getByRole('button', {name: 'Cancel'}).click();

    // default unchanged
    await expect(makeDefaultModal).toBeHidden();
    // TODO: test that no default was set
  });

  test('set default failure displays toast', async ({page}) => {
    await expect(page.locator('.project-column')).toBeHidden();
    await tryCreateColumn(page);
    await tryCreateColumn(page);
    await expect(page.locator('.project-column')).toHaveCount(2);
    const testColumn = page.locator('.project-column').last();

    const pathname = await page.evaluate(() => window.location.pathname);
    await page.route(`${pathname}/*/default`, (route) => route.abort('internetdisconnected'));

    const makeDefaultModal = page.locator('dialog[id^="default-project-column-modal-"]').last();
    await expect(makeDefaultModal).toBeHidden();
    await testColumn.getByRole('combobox').click(); // TODO: label, and improved dropdown?
    await testColumn.getByRole('option', {name: 'Set default'}).click();
    await expect(makeDefaultModal).toBeVisible();
    await makeDefaultModal.getByRole('button', {name: 'Confirm'}).click();

    // default failed
    await expect(makeDefaultModal).toBeVisible();
    await expect(page.locator('.toast-body', {hasText: 'Network error'})).toBeVisible();
  });

  // TODO: project delete behaves
  // TODO: toast when response goes through but it's 4xx or 5xx
});
