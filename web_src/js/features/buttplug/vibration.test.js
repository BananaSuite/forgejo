import {afterEach, describe, expect, test, vi} from 'vitest';
import {createVibration} from './vibration.js';

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('Buttplug vibration', () => {
  test('stops a pulse after 180 ms', async () => {
    vi.useFakeTimers();
    vi.stubGlobal('document', {hidden: false});
    const device = {runOutput: vi.fn().mockResolvedValue(), stop: vi.fn().mockResolvedValue()};
    const vibration = createVibration();

    await vibration.start(device, 'vibrate');
    expect(device.runOutput).toHaveBeenCalledWith('vibrate');
    await vi.advanceTimersByTimeAsync(180);
    expect(device.stop).toHaveBeenCalledOnce();
  });

  test('ignores hidden pages and repeated pulses during cooldown', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(10000);
    vi.stubGlobal('document', {hidden: true});
    const device = {runOutput: vi.fn().mockResolvedValue(), stop: vi.fn().mockResolvedValue()};
    const vibration = createVibration();

    await vibration.start(device, 'vibrate');
    expect(device.runOutput).not.toHaveBeenCalled();
    document.hidden = false;
    await vibration.start(device, 'vibrate');
    await vibration.start(device, 'vibrate');
    await vi.advanceTimersByTimeAsync(180);
    await vibration.start(device, 'vibrate');
    expect(device.runOutput).toHaveBeenCalledOnce();
    await vi.advanceTimersByTimeAsync(1500);
    await vibration.start(device, 'vibrate');
    expect(device.runOutput).toHaveBeenCalledTimes(2);
    await vibration.stop();
  });

  test('cancels an output that finishes after disconnect', async () => {
    vi.useFakeTimers();
    vi.stubGlobal('document', {hidden: false});
    let finish;
    const device = {
      runOutput: vi.fn(() => new Promise((resolve) => {finish = resolve;})),
      stop: vi.fn().mockResolvedValue(),
    };
    const vibration = createVibration();

    const sending = vibration.start(device, 'vibrate');
    await vibration.stop();
    finish();
    await sending;
    expect(device.stop).toHaveBeenCalledTimes(2);
  });

  test('handles device errors', async () => {
    vi.useFakeTimers();
    vi.stubGlobal('document', {hidden: false});
    const error = vi.spyOn(console, 'error').mockImplementation(() => {});
    const device = {runOutput: () => {throw new Error('output failed');}, stop: () => {throw new Error('stop failed');}};
    const vibration = createVibration();

    await expect(vibration.start(device, 'vibrate')).resolves.toBeUndefined();
    expect(error).toHaveBeenCalledTimes(2);
  });
});
