import {watchForgejoEvents} from './events.js';
import {createVibration} from './vibration.js';

/**
 * @typedef {{index: number, name: string}} DeviceSelection
 */

/**
 * @typedef {Object} ButtplugPreferences
 * @property {boolean} [enabled] - Reconnect during this browser session.
 * @property {string} [address] - Intiface WebSocket address.
 * @property {DeviceSelection | null} [device] - Selected device identity.
 * @property {boolean} [notifications] - React to unread-count increases.
 * @property {boolean} [stopwatches] - React to stopwatch starts.
 */

/**
 * Get the session key for a Forgejo user.
 * @param {string} userId - Forgejo user ID.
 * @returns {string} Storage key.
 */
const sessionKey = (userId) => `forgejo-buttplug:${userId}`;
/** Default Intiface WebSocket address. */
const defaultAddress = 'ws://127.0.0.1:12345';

/**
 * Read a user's session preferences, falling back to an empty record.
 * @param {string} userId - Forgejo user ID.
 * @returns {ButtplugPreferences} Stored preferences.
 */
export function readPreferences(userId) {
  try {
    return JSON.parse(sessionStorage.getItem(sessionKey(userId))) || {};
  } catch {
    return {};
  }
}

/**
 * Store session preferences when browser storage is available.
 * @param {string} userId - Forgejo user ID.
 * @param {ButtplugPreferences} preferences - Settings to save.
 * @returns {void}
 */
function savePreferences(userId, preferences) {
  try {
    sessionStorage.setItem(sessionKey(userId), JSON.stringify(preferences));
  } catch {}
}

/**
 * Set up Buttplug controls and restore the current user's connection.
 * Does nothing when the user-menu control is absent.
 * @returns {void}
 */
export function initButtplug() {
  const panel = document.getElementById('buttplug-panel');
  const quickStop = document.getElementById('buttplug-quick-stop');
  if (!quickStop) return;
  const userId = quickStop.dataset.userId;
  const controls = panel && Object.fromEntries([
    'address', 'connect', 'disconnect', 'scan', 'test', 'device', 'status', 'notifications', 'stopwatches',
  ].map((name) => [name, panel.querySelector(`#buttplug-${name}`)]));
  const vibration = createVibration();
  let preferences = readPreferences(userId);
  let connection = null;

  /**
   * Find the saved device only if its index, name, and capability match.
   * @returns {import('buttplug').ButtplugClientDevice | null} Selected device.
   */
  function selectedDevice() {
    const {client, api} = connection || {};
    const selection = preferences.device;
    if (!client?.connected || !selection) return null;
    const device = client.devices.get(selection.index);
    return device?.name === selection.name && device.hasOutput(api.OutputType.Vibrate) ? device : null;
  }

  /**
   * Update the controls for the current connection state.
   * @param {string} [status] - Optional localized status key.
   * @returns {void}
   */
  function render(status) {
    const connected = Boolean(connection?.client?.connected);
    const connecting = Boolean(connection && !connected);
    quickStop?.classList.toggle('tw-hidden', !connection);
    if (!controls) return;
    controls.address.disabled = Boolean(connection);
    controls.connect.disabled = Boolean(connection);
    controls.disconnect.disabled = !connection;
    controls.scan.disabled = !connected;
    controls.device.disabled = !connected;
    controls.test.disabled = !selectedDevice();
    controls.status.textContent = controls.status.dataset[status || (connected ? 'connected' : connecting ? 'connecting' : 'disconnected')];
  }

  /**
   * List vibration-capable devices and restore the selected option.
   * @returns {void}
   */
  function refreshDevices() {
    if (!controls) return;
    const {client, api} = connection || {};
    const devices = client?.connected ? [...client.devices.values()].filter((device) => device.hasOutput(api.OutputType.Vibrate)) : [];
    controls.device.replaceChildren(new Option(controls.device.options[0].textContent, ''));
    for (const device of devices) {
      controls.device.add(new Option(device.displayName || device.name, String(device.index)));
    }
    controls.device.value = selectedDevice() ? String(preferences.device.index) : '';
    render();
  }

  /**
   * Pulse the selected device at 20% intensity.
   * @returns {Promise<void>}
   */
  async function vibrate() {
    const device = selectedDevice();
    if (device) await vibration.start(device, connection.api.DeviceOutput.Vibrate.percent(0.2));
  }

  /**
   * Subscribe to enabled events when a selected device is connected.
   * @returns {void}
   */
  function watchEvents() {
    if (!connection?.client?.connected) return;
    connection.unsubscribe?.();
    connection.unsubscribe = null;
    if (!selectedDevice()) return;
    const notification = preferences.notifications ? vibrate : null;
    const stopwatch = preferences.stopwatches && window.config.enableTimeTracking ? vibrate : null;
    if (notification || stopwatch) connection.unsubscribe = watchForgejoEvents({notification, stopwatch});
  }

  /**
   * Stop vibration, unsubscribe, and disconnect the active attempt.
   * @param {boolean} [resume=false] - Reconnect after page navigation.
   * @returns {Promise<void>}
   */
  async function disconnect(resume = false) {
    const previous = connection;
    connection = null;
    previous?.unsubscribe?.();
    preferences = {...preferences, enabled: resume && preferences.enabled};
    savePreferences(userId, preferences);
    render('disconnected');
    await vibration.stop();
    try {
      await previous?.connector?.disconnect();
    } catch (error) {
      console.error('Buttplug: disconnect failed', error);
    }
  }

  /**
   * Load Buttplug and connect to the configured Intiface server.
   * @returns {Promise<void>}
   */
  async function connect() {
    if (connection) return;
    const address = controls?.address.value || preferences.address || defaultAddress;
    if (!/^wss?:\/\/[^\s/]+/.test(address)) {
      render('invalidAddress');
      return;
    }
    const attempt = {client: null, connector: null, api: null, unsubscribe: null};
    connection = attempt;
    render('connecting');
    try {
      const api = await import('buttplug');
      if (connection !== attempt) return;
      const client = new api.ButtplugClient('Forgejo');
      const connector = new api.ButtplugBrowserWebsocketClientConnector(address);
      Object.assign(attempt, {api, client, connector});
      await client.connect(connector);
      if (connection !== attempt) {
        await connector.disconnect();
        return;
      }
      client.on('deviceadded', () => {
        if (connection !== attempt) return;
        refreshDevices();
        watchEvents();
      });
      client.on('deviceremoved', () => {
        if (connection !== attempt) return;
        void vibration.stop();
        if (!selectedDevice()) {
          preferences = {...preferences, device: null};
          savePreferences(userId, preferences);
        }
        refreshDevices();
        watchEvents();
      });
      client.on('disconnect', () => {
        if (connection === attempt) void disconnect();
      });
      preferences = {...preferences, enabled: true, address};
      savePreferences(userId, preferences);
      refreshDevices();
      watchEvents();
      render('connected');
    } catch (error) {
      console.error('Buttplug: connection failed', error);
      if (connection === attempt) {
        await disconnect();
        render('failed');
      }
    }
  }

  quickStop?.querySelector('button')?.addEventListener('click', () => void disconnect());
  if (controls) {
    controls.address.value = preferences.address || defaultAddress;
    controls.notifications.checked = Boolean(preferences.notifications);
    controls.stopwatches.checked = Boolean(preferences.stopwatches && window.config.enableTimeTracking);
    controls.connect.addEventListener('click', () => void connect());
    controls.disconnect.addEventListener('click', () => void disconnect());
    controls.scan.addEventListener('click', async () => {
      try {
        await connection?.client?.startScanning();
      } catch (error) {
        console.error('Buttplug: scan failed', error);
        render('scanFailed');
      }
    });
    controls.test.addEventListener('click', () => void vibrate());
    controls.device.addEventListener('change', () => {
      void vibration.stop();
      const device = connection?.client?.devices.get(Number(controls.device.value));
      preferences = {...preferences, device: controls.device.value && device ? {index: device.index, name: device.name} : null};
      savePreferences(userId, preferences);
      watchEvents();
      render();
    });
    for (const name of ['notifications', 'stopwatches']) {
      controls[name].addEventListener('change', () => {
        preferences = {...preferences, [name]: controls[name].checked};
        savePreferences(userId, preferences);
        watchEvents();
      });
    }
  }
  document.getElementById('logout-user-action')?.addEventListener('submit', () => void disconnect());
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) void vibration.stop();
  });
  window.addEventListener('pagehide', () => void disconnect(true));
  window.addEventListener('pageshow', (event) => {
    if (event.persisted && preferences.enabled) void connect();
  });
  render();
  if (preferences.enabled) void connect();
}
