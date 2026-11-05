// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
import {flushPromises, mount} from '@vue/test-utils';
import DashboardRepoList from './DashboardRepoList.vue';

const repos = Array.from({length: 5}, (_, i) => ({id: i + 1, full_name: `user1/repo${i + 1}`, link: `/user1/repo${i + 1}`}));

function mockResponse(data, {total = data.length, ok = true} = {}) {
  return {
    ok,
    headers: new Headers({'X-Total-Count': String(total)}),
    json: () => Promise.resolve({data: data.map((repository) => ({repository}))}),
  };
}

function respondWithPage(request) {
  const page = Number(new URL(request.url, 'http://localhost').searchParams.get('page'));
  request.resolve(mockResponse(repos.slice((page - 1) * 2, page * 2), {total: repos.length}));
}

function repoNames(wrapper) {
  return wrapper.findAll('.repo-owner-name-list li').map((li) => li.text());
}

// Search requests stay pending until the test resolves them, so the
// component's state can be inspected while it is loading.
async function mountRepoList() {
  window.history.replaceState({}, '', '/');
  window.config.pageData.dashboardRepoList = {
    searchLimit: 2,
    uid: 1,
    teamId: 0,
    isMirrorsEnabled: false,
    textMyRepos: '',
    textSearchRepos: '',
    textFilter: '',
    textShowArchived: '',
    textShowPrivate: '',
    textAll: '',
    textSources: '',
    textForks: '',
    textCollaborative: '',
    textFirstPage: '',
    textPreviousPage: '',
    textNextPage: 'Next',
    textLastPage: 'Last',
  };

  const requests = [];
  vi.spyOn(global, 'fetch').mockImplementation((url) => {
    if (url.includes('count_only=1')) return Promise.resolve(mockResponse([], {total: 42}));
    return new Promise((resolve) => requests.push({url, resolve}));
  });

  const wrapper = mount(DashboardRepoList, {
    global: {stubs: {'overflow-menu': {template: '<div><slot/></div>'}}},
  });
  await flushPromises();
  respondWithPage(requests.shift());
  await flushPromises();
  return {wrapper, requests};
}

test('keeps showing the current repos while a new filter is loading', async () => {
  const {wrapper, requests} = await mountRepoList();

  await wrapper.findAll('.repos-filter .item')[1].trigger('click');
  await flushPromises();
  expect(requests[0].url).toContain('mode=source');
  expect(repoNames(wrapper)).toEqual(['user1/repo1', 'user1/repo2']);

  requests[0].resolve(mockResponse([{id: 10, full_name: 'user1/source', link: ''}]));
  await flushPromises();
  expect(repoNames(wrapper)).toEqual(['user1/source']);
});

test('keeps showing the current repos and pagination while a new page is loading', async () => {
  const {wrapper, requests} = await mountRepoList();

  await wrapper.get('.pagination .item[title="Next"]').trigger('click');
  await flushPromises();
  expect(repoNames(wrapper)).toEqual(['user1/repo1', 'user1/repo2']);
  expect(wrapper.find('.pagination').exists()).toBe(true);
  expect(wrapper.get('.repos-filter .item.active .label').text()).toBe('5');

  respondWithPage(requests[0]);
  await flushPromises();
  expect(repoNames(wrapper)).toEqual(['user1/repo3', 'user1/repo4']);
});

test('ignores page changes while a page is loading', async () => {
  const {wrapper, requests} = await mountRepoList();

  await wrapper.get('.pagination .item[title="Next"]').trigger('click');
  await wrapper.get('.pagination .item[title="Next"]').trigger('click');
  await wrapper.get('.pagination .item[title="Last"]').trigger('click');
  await flushPromises();
  expect(requests).toHaveLength(1);

  respondWithPage(requests[0]);
  await flushPromises();
  expect(wrapper.get('.pagination .active.item').text()).toBe('2');
  expect(repoNames(wrapper)).toEqual(['user1/repo3', 'user1/repo4']);
});

test('clears the repos and count when the search fails', async () => {
  const {wrapper, requests} = await mountRepoList();

  await wrapper.get('.pagination .item[title="Next"]').trigger('click');
  await flushPromises();
  requests[0].resolve(mockResponse([], {ok: false}));
  await flushPromises();

  expect(repoNames(wrapper)).toEqual([]);
  expect(wrapper.find('.pagination').exists()).toBe(false);
  expect(wrapper.get('.repos-filter .item.active .label').text()).toBe('');
});

test('ArrowUp on the first repo selects the last repo of the previous page', async () => {
  const {wrapper, requests} = await mountRepoList();
  const input = wrapper.get('input[type="search"]');

  await input.trigger('keydown', {key: 'ArrowRight'});
  respondWithPage(requests.shift());
  await flushPromises();
  expect(wrapper.get('li.active').text()).toBe('user1/repo3');

  await input.trigger('keydown', {key: 'ArrowUp'});
  respondWithPage(requests.shift());
  await flushPromises();
  expect(wrapper.get('li.active').text()).toBe('user1/repo2');
});
