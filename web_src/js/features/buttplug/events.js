/**
 * Check for an increase in unread notifications. The first count sets a baseline.
 * @param {number | undefined} previous - Last unread count.
 * @param {number} next - New unread count.
 * @returns {boolean} Whether the count increased.
 */
export function notificationCountIncreased(previous, next) {
  return Number.isSafeInteger(previous) && Number.isSafeInteger(next) && next > previous;
}

/**
 * Detect a new active stopwatch from successive issue identifiers.
 * @param {string | null | undefined} previous - Previous active issue.
 * @param {string | null} next - Current active issue.
 * @returns {boolean} Whether a stopwatch started on a different issue.
 */
export function stopwatchStarted(previous, next) {
  return previous !== undefined && next !== null && next !== previous;
}

/**
 * Identify the first active stopwatch in a Forgejo response.
 * @param {Array<{repo_owner_name: string, repo_name: string, issue_index: number}> | null} watches - Active stopwatches.
 * @returns {string | null} Issue identifier, or null if no valid stopwatch exists.
 */
export function stopwatchIssue(watches) {
  const watch = Array.isArray(watches) ? watches[0] : null;
  if (!watch) return null;
  const {repo_owner_name: owner, repo_name: repo, issue_index: index} = watch;
  if (typeof owner !== 'string' || typeof repo !== 'string' || !Number.isSafeInteger(index)) return null;
  return `${owner}/${repo}#${index}`;
}

/**
 * Watch selected notification and stopwatch events.
 * Uses the shared worker when available and polling otherwise.
 * @param {{notification: (() => void) | null, stopwatch: (() => void) | null}} handlers - Enabled callbacks.
 * @returns {() => void} Stop watching and discard subsequent updates.
 */
export function watchForgejoEvents({notification, stopwatch}) {
  const {appSubUrl, assetVersionEncoded, enableTimeTracking, notificationSettings} = window.config;
  const countText = document.querySelector('.notification_count')?.textContent;
  let previousCount = countText === undefined ? undefined : Number(countText);
  const activeStopwatch = document.querySelector('.active-stopwatch-trigger:not(.tw-hidden)');
  let previousIssue = activeStopwatch ? document.querySelector('.stopwatch-issue')?.textContent?.trim() : null;
  let worker;
  let timer;
  let closed = false;

  /**
   * Update the unread-count baseline and invoke the notification callback.
   * @param {number} next - New unread count.
   * @returns {void}
   */
  function receiveCount(next) {
    if (!Number.isSafeInteger(next) || next < 0) return;
    if (notification && notificationCountIncreased(previousCount, next)) notification();
    previousCount = next;
  }

  /**
   * Update the active issue and invoke the stopwatch callback for a new start.
   * @param {string | null} next - Current active issue.
   * @returns {void}
   */
  function receiveStopwatch(next) {
    if (stopwatch && stopwatchStarted(previousIssue, next)) stopwatch();
    previousIssue = next;
  }

  /**
   * Handle messages from Forgejo's notification shared worker.
   * @param {{data: {type: string, data?: string}}} event - Worker message.
   * @returns {void}
   */
  function onMessage({data}) {
    if (closed) return;
    try {
      if (data.type === 'notification-count') {
        receiveCount(JSON.parse(data.data).Count);
      } else if (data.type === 'stopwatches' && enableTimeTracking) {
        receiveStopwatch(stopwatchIssue(JSON.parse(data.data)));
      } else if (data.type === 'no-event-source') {
        worker?.port.close();
        worker = null;
        void poll();
      }
    } catch (error) {
      console.error('Buttplug: event parsing failed', error);
    }
  }

  /**
   * Fetch enabled event types and schedule the next poll until unsubscribed.
   * @returns {Promise<void>}
   */
  async function poll() {
    if (closed) return;
    try {
      const requests = [];
      if (notification) requests.push(fetch(`${appSubUrl}/notifications/new`).then(async (response) => {
        if (!response.ok) throw new Error(`Forgejo returned HTTP ${response.status}`);
        const count = await response.json();
        if (!closed) receiveCount(count.new);
      }));
      if (stopwatch && enableTimeTracking) requests.push(fetch(`${appSubUrl}/user/stopwatches`).then(async (response) => {
        if (!response.ok) throw new Error(`Forgejo returned HTTP ${response.status}`);
        const watches = await response.json();
        if (!closed) receiveStopwatch(stopwatchIssue(watches));
      }));
      await Promise.all(requests);
    } catch (error) {
      console.error('Buttplug: polling failed', error);
    } finally {
      const timeout = notificationSettings.MinTimeout;
      const interval = Number.isFinite(timeout) ? Math.max(timeout, 15000) : 15000;
      if (!closed) timer = setTimeout(poll, interval);
    }
  }

  try {
    if (notificationSettings.EventSourceUpdateTime <= 0 || !window.SharedWorker || !window.EventSource) {
      void poll();
    } else {
      worker = new SharedWorker(`${__webpack_public_path__}js/eventsource.sharedworker.js?v=${assetVersionEncoded}`, 'notification-worker');
      worker.port.addEventListener('message', onMessage);
      worker.port.start();
      worker.port.postMessage({type: 'start', url: `${window.location.origin}${appSubUrl}/user/events`});
    }
  } catch (error) {
    console.error('Buttplug: event subscription failed', error);
    worker?.port.close();
    worker = null;
    void poll();
  }

  return () => {
    closed = true;
    clearTimeout(timer);
    if (worker) {
      worker.port.postMessage({type: 'close'});
      worker.port.close();
    }
  };
}
