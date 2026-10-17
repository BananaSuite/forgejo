/**
 * Describe a virtual device with one 20-step vibration motor.
 * @param {number} index - Device index.
 * @param {string} name - Device name.
 * @returns {Object} Buttplug v4 device description.
 */
const deviceInfo = (index, name) => ({
  DeviceName: name,
  DeviceIndex: index,
  DeviceMessageTimingGap: 0,
  DeviceFeatures: {
    0: {
      FeatureIndex: 0,
      FeatureDescription: 'Motor',
      Output: {Vibrate: {Value: [0, 20]}},
    },
  },
});

/**
 * @typedef {Object} VirtualButtplugServer
 * @property {Function} WebSocket - WebSocket replacement constructor.
 * @property {Array<{index: number, value: number}>} commands - Motor commands in order.
 * @property {(index?: number) => number | undefined} level - Current motor level.
 * @property {() => number} connected - Number of open sockets.
 * @property {(index: number, name?: string) => void} addDevice - Add a motor.
 * @property {(index: number) => void} removeDevice - Remove a motor.
 */

/**
 * Create an in-memory Buttplug v4 server and WebSocket replacement.
 * @returns {VirtualButtplugServer} Server controls and recorded commands.
 */
export function createVirtualButtplugServer() {
  const devices = new Map([[0, deviceInfo(0, 'Virtual Buttplug')]]);
  const levels = new Map([[0, 0]]);
  const sockets = new Set();
  const commands = [];

  /**
   * Return the current device list as a protocol message.
   * @param {number} [id=0] - Request ID.
   * @returns {Object} DeviceList response.
   */
  function deviceList(id = 0) {
    return {DeviceList: {Id: id, Devices: Object.fromEntries(devices)}};
  }

  /**
   * Send the updated device list to connected clients.
   * @returns {void}
   */
  function notifyDevices() {
    for (const socket of sockets) socket.deliver(deviceList());
  }

  /**
   * Process one client message and update virtual motor levels.
   * @param {Object} message - Buttplug v4 request.
   * @returns {Object} Protocol response.
   */
  function handle(message) {
    const [name, request] = Object.entries(message)[0];
    const {Id: id} = request;
    switch (name) {
      case 'RequestServerInfo':
        return {ServerInfo: {Id: id, ServerName: 'Virtual Buttplug', ProtocolVersionMajor: 4, ProtocolVersionMinor: 0, MaxPingTime: 0}};
      case 'RequestDeviceList':
        return deviceList(id);
      case 'StartScanning':
      case 'StopScanning':
        return {Ok: {Id: id}};
      case 'OutputCmd': {
        const {DeviceIndex: index, FeatureIndex: feature, Command: command} = request;
        const rawValue = command.Vibrate?.Value;
        // Buttplug 4.0.2 sends scalar values as one-element arrays.
        const value = Array.isArray(rawValue) ? rawValue[0] : rawValue;
        if (!devices.has(index) || feature !== 0 || !Number.isInteger(value) || value < 0 || value > 20) break;
        levels.set(index, value);
        commands.push({index, value});
        return {Ok: {Id: id}};
      }
      case 'StopCmd': {
        const indices = request.DeviceIndex === undefined ? [...devices.keys()] : [request.DeviceIndex];
        if (indices.some((index) => !devices.has(index))) break;
        for (const index of indices) {
          levels.set(index, 0);
          commands.push({index, value: 0});
        }
        return {Ok: {Id: id}};
      }
    }
    return {Error: {Id: id, ErrorMessage: `Unsupported ${name}`, ErrorCode: 3}};
  }

  /** WebSocket replacement connected to the virtual server. */
  class VirtualWebSocket {
    listeners = new Map();
    closed = false;

    /** Register a socket and emit its open event. */
    constructor() {
      sockets.add(this);
      queueMicrotask(() => this.emit('open'));
    }

    /**
     * @param {string} type - Event name.
     * @param {Function} listener - Event handler.
     * @returns {void}
     */
    addEventListener(type, listener) {
      if (!this.listeners.has(type)) this.listeners.set(type, new Set());
      this.listeners.get(type).add(listener);
    }

    /**
     * @param {string} type - Event name.
     * @param {Function} listener - Event handler.
     * @returns {void}
     */
    removeEventListener(type, listener) {
      this.listeners.get(type)?.delete(listener);
    }

    /**
     * Dispatch a socket event to registered listeners.
     * @param {string} type - Event name.
     * @param {Object} [event] - Event payload.
     * @returns {void}
     */
    emit(type, event = {}) {
      for (const listener of this.listeners.get(type) || []) listener(event);
    }

    /**
     * Deliver a protocol response on the next microtask.
     * @param {Object} message - Protocol response.
     * @returns {void}
     */
    deliver(message) {
      queueMicrotask(() => {
        if (!this.closed) this.emit('message', {data: JSON.stringify([message])});
      });
    }

    /**
     * Handle a JSON array of protocol requests.
     * @param {string} payload - Serialized requests.
     * @returns {void}
     */
    send(payload) {
      for (const message of JSON.parse(payload)) this.deliver(handle(message));
    }

    /** Close the socket and notify listeners. */
    close() {
      if (this.closed) return;
      this.closed = true;
      sockets.delete(this);
      this.emit('close');
    }
  }

  return {
    WebSocket: VirtualWebSocket,
    commands,
    level: (index = 0) => levels.get(index),
    connected: () => sockets.size,
    /**
     * Add a virtual motor and notify connected clients.
     * @param {number} index - Device index.
     * @param {string} [name] - Device name.
     * @returns {void}
     */
    addDevice(index, name = 'Virtual Buttplug') {
      devices.set(index, deviceInfo(index, name));
      levels.set(index, 0);
      notifyDevices();
    },
    /**
     * Remove a virtual motor and notify connected clients.
     * @param {number} index - Device index.
     * @returns {void}
     */
    removeDevice(index) {
      devices.delete(index);
      levels.delete(index);
      notifyDevices();
    },
  };
}
