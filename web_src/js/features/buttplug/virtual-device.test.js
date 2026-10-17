import {afterEach, describe, expect, test, vi} from 'vitest';
import {ButtplugBrowserWebsocketClientConnector, ButtplugClient, DeviceOutput, OutputType} from 'buttplug/dist/web/buttplug.mjs';
import {watchForgejoEvents} from './events.js';
import {createVibration} from './vibration.js';
import {createVirtualButtplugServer} from './test/virtual-device.js';

let connector;
let vibration;

afterEach(async () => {
  await vibration?.stop();
  await connector?.disconnect();
  connector = null;
  vibration = null;
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

/**
 * Connect a real Buttplug client to the virtual server.
 * @param {ReturnType<typeof createVirtualButtplugServer>} server - Virtual server.
 * @returns {Promise<import('buttplug').ButtplugClient>} Connected client.
 */
async function connect(server) {
  vi.stubGlobal('WebSocket', server.WebSocket);
  const client = new ButtplugClient('Forgejo tests');
  connector = new ButtplugBrowserWebsocketClientConnector('ws://127.0.0.1:12345');
  await client.connect(connector);
  return client;
}

describe('virtual Buttplug device', () => {
  test('discovers devices and starts scanning', async () => {
    const server = createVirtualButtplugServer();
    server.addDevice(1, 'Second vibrator');
    const client = await connect(server);

    expect([...client.devices.keys()]).toEqual([0, 1]);
    expect(client.devices.get(0).hasOutput(OutputType.Vibrate)).toBe(true);
    await client.startScanning();
    expect(client.isScanning).toBe(true);
    expect(server.connected()).toBe(1);
  });

  test('converts a 20% pulse to four steps and stops after 180 ms', async () => {
    const server = createVirtualButtplugServer();
    const client = await connect(server);
    vibration = createVibration();
    vi.stubGlobal('document', {hidden: false});
    vi.useFakeTimers();

    await vibration.start(client.devices.get(0), DeviceOutput.Vibrate.percent(0.2));
    expect(server.level()).toBe(4);
    await vi.advanceTimersByTimeAsync(179);
    expect(server.level()).toBe(4);
    await vi.advanceTimersByTimeAsync(1);
    expect(server.level()).toBe(0);
    expect(server.commands).toEqual([{index: 0, value: 4}, {index: 0, value: 0}]);
  });

  test('stops only the selected device', async () => {
    const server = createVirtualButtplugServer();
    server.addDevice(1, 'Second vibrator');
    const client = await connect(server);
    vibration = createVibration();
    vi.stubGlobal('document', {hidden: false});
    vi.useFakeTimers();

    await client.devices.get(1).runOutput(DeviceOutput.Vibrate.percent(0.5));
    await vibration.start(client.devices.get(0), DeviceOutput.Vibrate.percent(0.2));
    await vi.advanceTimersByTimeAsync(180);

    expect(server.level(0)).toBe(0);
    expect(server.level(1)).toBe(10);
    await client.devices.get(1).stop();
  });

  test('handles device removal during a pulse', async () => {
    const server = createVirtualButtplugServer();
    const client = await connect(server);
    const error = vi.spyOn(console, 'error').mockImplementation(() => {});
    vibration = createVibration();
    vi.stubGlobal('document', {hidden: false});
    vi.useFakeTimers();

    await vibration.start(client.devices.get(0), DeviceOutput.Vibrate.percent(0.2));
    server.removeDevice(0);
    await vi.advanceTimersByTimeAsync(180);

    expect(client.connected).toBe(true);
    expect(client.devices.size).toBe(0);
    expect(error).toHaveBeenCalledOnce();
    server.addDevice(1);
    await vi.advanceTimersByTimeAsync(0);
    expect(client.devices.has(1)).toBe(true);
  });

  test('routes Forgejo notification and stopwatch events to the virtual device', async () => {
    const server = createVirtualButtplugServer();
    const client = await connect(server);
    vibration = createVibration();
    let receive;
    class Worker {
      port = {
        addEventListener: (type, listener) => {if (type === 'message') receive = listener;},
        start() {},
        postMessage() {},
        close() {},
      };
    }
    vi.stubGlobal('__webpack_public_path__', '/assets/');
    vi.stubGlobal('window', {
      SharedWorker: Worker,
      EventSource: class {},
      location: {origin: 'https://forgejo.test'},
      config: {
        appSubUrl: '', assetVersionEncoded: 'test', enableTimeTracking: true,
        notificationSettings: {EventSourceUpdateTime: 1000, MinTimeout: 1000},
      },
    });
    vi.stubGlobal('SharedWorker', Worker);
    vi.stubGlobal('document', {
      hidden: false,
      querySelector: (selector) => ({
        '.notification_count': {textContent: '2'},
        '.active-stopwatch-trigger:not(.tw-hidden)': null,
      })[selector],
    });
    vi.useFakeTimers();
    const pulse = () => void vibration.start(client.devices.get(0), DeviceOutput.Vibrate.percent(0.2));
    const unsubscribe = watchForgejoEvents({notification: pulse, stopwatch: pulse});
    const send = (type, data) => receive({data: {type, data: JSON.stringify(data)}});
    try {
      send('notification-count', {Count: 2});
      expect(server.level()).toBe(0);
      send('notification-count', {Count: 3});
      await vi.advanceTimersByTimeAsync(0);
      expect(server.level()).toBe(4);
      send('notification-count', {Count: 3});
      await vi.advanceTimersByTimeAsync(180);
      expect(server.commands).toHaveLength(2);
      await vi.advanceTimersByTimeAsync(1500);
      send('stopwatches', [{repo_owner_name: 'owner', repo_name: 'repo', issue_index: 1}]);
      await vi.advanceTimersByTimeAsync(0);
      expect(server.level()).toBe(4);
      await vi.advanceTimersByTimeAsync(180);
      expect(server.commands).toEqual([
        {index: 0, value: 4}, {index: 0, value: 0},
        {index: 0, value: 4}, {index: 0, value: 0},
      ]);
    } finally {
      unsubscribe();
    }
  });
});
