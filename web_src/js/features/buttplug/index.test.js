import {afterEach, describe, expect, test, vi} from 'vitest';
import {readPreferences} from './index.js';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('Buttplug session preferences', () => {
  test('keeps preferences separate for different Forgejo users', () => {
    const entries = new Map([['forgejo-buttplug:1', JSON.stringify({enabled: true})]]);
    vi.stubGlobal('sessionStorage', {getItem: (key) => entries.get(key) || null});
    expect(readPreferences('1')).toEqual({enabled: true});
    expect(readPreferences('2')).toEqual({});
  });

  test('treats unavailable or invalid session data as disabled', () => {
    vi.stubGlobal('sessionStorage', {getItem: () => '{invalid'});
    expect(readPreferences('1')).toEqual({});
    vi.stubGlobal('sessionStorage', {getItem: () => {throw new Error('blocked');}});
    expect(readPreferences('1')).toEqual({});
  });
});

test('requires device and event selection before watching Forgejo events', async () => {
  const {createVirtualButtplugServer} = await import('./test/virtual-device.js');
  const {initButtplug} = await import('./index.js');
  const server = createVirtualButtplugServer();
  const entries = new Map();
  vi.stubGlobal('sessionStorage', {
    getItem: (key) => entries.get(key) || null,
    setItem: (key, value) => entries.set(key, value),
  });
  vi.stubGlobal('WebSocket', server.WebSocket);
  vi.stubGlobal('Option', class {
    constructor(textContent, value) {Object.assign(this, {textContent, value});}
  });

  class Control {
    listeners = new Map();
    options = [{textContent: 'Choose a device'}];
    dataset = {connected: 'Connected', disconnected: 'Disconnected', connecting: 'Connecting'};
    value = '';
    checked = false;
    addEventListener(type, listener) {this.listeners.set(type, listener);}
    trigger(type) {this.listeners.get(type)?.();}
    replaceChildren(...options) {this.options = options;}
    add(option) {this.options.push(option);}
  }

  const names = ['address', 'connect', 'disconnect', 'scan', 'test', 'device', 'status', 'notifications', 'stopwatches'];
  const controls = Object.fromEntries(names.map((name) => [name, new Control()]));
  const panel = {querySelector: (selector) => controls[selector.slice('#buttplug-'.length)]};
  const stop = new Control();
  const quickStop = {
    dataset: {userId: '42'},
    classList: {toggle: vi.fn()},
    querySelector: () => stop,
  };
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
    SharedWorker: Worker, EventSource: class {}, location: {origin: 'https://forgejo.test'},
    addEventListener() {},
    config: {
      appSubUrl: '', assetVersionEncoded: 'test', enableTimeTracking: true,
      notificationSettings: {EventSourceUpdateTime: 1000, MinTimeout: 15000},
    },
  });
  vi.stubGlobal('document', {
    hidden: false,
    addEventListener() {},
    getElementById: (id) => ({
      'buttplug-panel': panel,
      'buttplug-quick-stop': quickStop,
    })[id],
    querySelector: (selector) => selector === '.notification_count' ? {textContent: '2'} : null,
  });

  initButtplug();
  expect(server.connected()).toBe(0);
  controls.connect.trigger('click');
  await vi.waitFor(() => expect(server.connected()).toBe(1));
  expect(receive).toBeUndefined();

  controls.device.value = '0';
  controls.device.trigger('change');
  expect(receive).toBeUndefined();

  controls.notifications.checked = true;
  controls.notifications.trigger('change');
  expect(receive).toBeTypeOf('function');
  receive({data: {type: 'notification-count', data: '{"Count":3}'}});
  await vi.waitFor(() => expect(server.level()).toBe(4));
  await vi.waitFor(() => expect(server.level()).toBe(0));

  const commandCount = server.commands.length;
  stop.trigger('click');
  await vi.waitFor(() => expect(server.connected()).toBe(0));
  receive({data: {type: 'notification-count', data: '{"Count":4}'}});
  expect(server.commands).toHaveLength(commandCount);
  expect(readPreferences('42').enabled).toBe(false);
  expect(readPreferences('43')).toEqual({});
});
