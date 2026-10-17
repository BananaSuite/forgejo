import {afterEach, describe, expect, test, vi} from 'vitest';
import {notificationCountIncreased, stopwatchIssue, stopwatchStarted, watchForgejoEvents} from './events.js';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('Buttplug event detection', () => {
  test('ignores the first count and unchanged or decreasing counts', () => {
    expect(notificationCountIncreased(undefined, 4)).toBe(false);
    expect(notificationCountIncreased(4, 4)).toBe(false);
    expect(notificationCountIncreased(4, 2)).toBe(false);
    expect(notificationCountIncreased(2, 5)).toBe(true);
    expect(notificationCountIncreased(2, NaN)).toBe(false);
  });

  test('recognizes starts, not repeated stopwatch snapshots', () => {
    expect(stopwatchStarted(undefined, 'owner/repo#1')).toBe(false);
    expect(stopwatchStarted(null, 'owner/repo#1')).toBe(true);
    expect(stopwatchStarted('owner/repo#1', 'owner/repo#1')).toBe(false);
    expect(stopwatchStarted('owner/repo#1', 'owner/repo#2')).toBe(true);
    expect(stopwatchStarted('owner/repo#1', null)).toBe(false);
    expect(stopwatchIssue([{repo_owner_name: 'owner', repo_name: 'repo', issue_index: 2}])).toBe('owner/repo#2');
    expect(stopwatchIssue([])).toBe(null);
    expect(stopwatchIssue([{repo_owner_name: 'owner', repo_name: 'repo', issue_index: '2'}])).toBe(null);
  });

  test('reads worker events and unsubscribes', () => {
    let messageListener;
    const postMessage = vi.fn();
    const close = vi.fn();
    class Worker {
      port = {
        addEventListener: (type, listener) => {
          if (type === 'message') messageListener = listener;
        },
        start: vi.fn(),
        postMessage,
        close,
      };
    }
    const notify = vi.fn();
    const stopwatch = vi.fn();
    vi.stubGlobal('__webpack_public_path__', '/assets/');
    vi.stubGlobal('window', {
      SharedWorker: Worker,
      EventSource: class {},
      location: {origin: 'https://forgejo.test'},
      config: {
        appSubUrl: '',
        assetVersionEncoded: 'test',
        enableTimeTracking: true,
        notificationSettings: {EventSourceUpdateTime: 1000, MinTimeout: 1000},
      },
    });
    vi.stubGlobal('SharedWorker', Worker);
    vi.stubGlobal('document', {querySelector: (selector) => ({
      '.notification_count': {textContent: '2'},
      '.active-stopwatch-trigger:not(.tw-hidden)': null,
    })[selector]});

    const unsubscribe = watchForgejoEvents({notification: notify, stopwatch});
    messageListener({data: {type: 'notification-count', data: JSON.stringify({Count: 2})}});
    messageListener({data: {type: 'notification-count', data: JSON.stringify({Count: 3})}});
    messageListener({data: {type: 'notification-count', data: JSON.stringify({Count: 3})}});
    messageListener({data: {type: 'stopwatches', data: JSON.stringify([{
      repo_owner_name: 'owner', repo_name: 'repo', issue_index: 1,
    }])}});
    messageListener({data: {type: 'stopwatches', data: JSON.stringify([{
      repo_owner_name: 'owner', repo_name: 'repo', issue_index: 1,
    }])}});
    expect(notify).toHaveBeenCalledTimes(1);
    expect(stopwatch).toHaveBeenCalledTimes(1);
    unsubscribe();
    messageListener({data: {type: 'notification-count', data: JSON.stringify({Count: 4})}});
    expect(notify).toHaveBeenCalledTimes(1);
    expect(postMessage).toHaveBeenCalledWith({type: 'close'});
    expect(close).toHaveBeenCalledTimes(1);
  });

  test('polls only while subscribed when the shared worker is unavailable', async () => {
    const notification = vi.fn();
    const stopwatch = vi.fn();
    const fetch = vi.fn()
      .mockResolvedValueOnce({ok: true, json: async () => ({new: 5})})
      .mockResolvedValueOnce({ok: true, json: async () => ([{
        repo_owner_name: 'owner', repo_name: 'repo', issue_index: 2,
      }])});
    vi.stubGlobal('fetch', fetch);
    vi.stubGlobal('window', {
      SharedWorker: undefined,
      EventSource: undefined,
      config: {
        appSubUrl: '/forgejo',
        enableTimeTracking: true,
        notificationSettings: {EventSourceUpdateTime: 0, MinTimeout: 20000},
      },
    });
    vi.stubGlobal('document', {querySelector: (selector) => ({
      '.notification_count': {textContent: '4'},
      '.active-stopwatch-trigger:not(.tw-hidden)': null,
    })[selector]});
    const unsubscribe = watchForgejoEvents({notification, stopwatch});
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(notification).toHaveBeenCalledOnce();
    expect(stopwatch).toHaveBeenCalledOnce();
    expect(fetch).toHaveBeenCalledTimes(2);
    unsubscribe();
  });
});

describe('Buttplug event selection', () => {
  test('does not invoke a disabled notification handler', () => {
    let receive;
    class Worker {
      port = {
        addEventListener: (type, listener) => {if (type === 'message') receive = listener;},
        start() {}, postMessage() {}, close() {},
      };
    }
    vi.stubGlobal('__webpack_public_path__', '/assets/');
    vi.stubGlobal('SharedWorker', Worker);
    vi.stubGlobal('window', {
      SharedWorker: Worker,
      EventSource: class {},
      location: {origin: 'https://forgejo.test'},
      config: {
        appSubUrl: '', assetVersionEncoded: 'test', enableTimeTracking: true,
        notificationSettings: {EventSourceUpdateTime: 1000, MinTimeout: 20000},
      },
    });
    vi.stubGlobal('document', {querySelector: () => null});
    const stopwatch = vi.fn();
    const unsubscribe = watchForgejoEvents({notification: null, stopwatch});
    receive({data: {type: 'notification-count', data: '{"Count":2}'}});
    receive({data: {type: 'notification-count', data: '{"Count":3}'}});
    receive({data: {type: 'stopwatches', data: JSON.stringify([
      {repo_owner_name: 'owner', repo_name: 'repo', issue_index: 1},
    ])}});
    expect(stopwatch).toHaveBeenCalledOnce();
    unsubscribe();
  });

  test('polls only for the selected event type', async () => {
    const fetch = vi.fn().mockResolvedValue({ok: true, json: async () => ({new: 2})});
    vi.stubGlobal('fetch', fetch);
    vi.stubGlobal('window', {
      SharedWorker: undefined,
      EventSource: undefined,
      config: {
        appSubUrl: '/forgejo', enableTimeTracking: true,
        notificationSettings: {EventSourceUpdateTime: 0, MinTimeout: 20000},
      },
    });
    vi.stubGlobal('document', {querySelector: () => null});
    const unsubscribe = watchForgejoEvents({notification: vi.fn(), stopwatch: null});
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(fetch).toHaveBeenCalledOnce();
    expect(fetch).toHaveBeenCalledWith('/forgejo/notifications/new');
    unsubscribe();
  });
});
