/** Length of a vibration pulse in milliseconds. */
const duration = 180;
/** Minimum time between pulse starts in milliseconds. */
const cooldown = 1500;

/**
 * @typedef {Object} VibrationDevice
 * @property {(output: unknown) => Promise<void>} runOutput - Send an output command.
 * @property {() => Promise<void>} stop - Stop this device.
 */

/**
 * Create a controller for one pulse at a time.
 * @returns {{start: (device: VibrationDevice, output: unknown) => Promise<void>, stop: () => Promise<void>}}
 *   Methods to start and cancel pulses.
 */
export function createVibration() {
  let active = null;
  let stopping = null;
  let lastPulse = -Infinity;

  /**
   * Cancel the active pulse and stop its device.
   * @returns {Promise<void>} Resolves after the stop command settles.
   */
  async function stop() {
    if (!active) return stopping;

    const pulse = active;
    active = null;
    clearTimeout(pulse.timer);

    const pending = Promise.resolve().then(() => pulse.device.stop()).catch((error) => {
      console.error('Buttplug: stop failed', error);
    });
    stopping = pending;
    try {
      await pending;
    } finally {
      if (stopping === pending) stopping = null;
    }
  }

  /**
   * Start a pulse unless the page is hidden, busy, or cooling down.
   * @param {VibrationDevice} device - Device to vibrate.
   * @param {unknown} output - Buttplug vibration command.
   * @returns {Promise<void>} Resolves after the output command settles.
   */
  async function start(device, output) {
    const now = Date.now();
    if (!device || document.hidden || active || stopping || now - lastPulse < cooldown) return;

    lastPulse = now;
    const pulse = {device, timer: setTimeout(() => void stop(), duration)};
    active = pulse;
    try {
      await device.runOutput(output);
      if (active !== pulse) await device.stop();
    } catch (error) {
      console.error('Buttplug: vibration failed', error);
      if (active === pulse) await stop();
    }
  }

  return {start, stop};
}
