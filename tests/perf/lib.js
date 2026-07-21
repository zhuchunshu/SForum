// Shared helpers for SForum public read-path k6 scripts.
// BASE_URL is the API origin only (e.g. http://127.0.0.1:8082).

export function baseUrl() {
  return (__ENV.BASE_URL || 'http://127.0.0.1:8081').replace(/\/$/, '');
}

export function api(path) {
  const p = path.startsWith('/') ? path : `/${path}`;
  return `${baseUrl()}/api/v1${p}`;
}

export function hotSlug() {
  return __ENV.HOT_SLUG || 'perf-hot-thread';
}

export function categorySlug() {
  return __ENV.CATEGORY_SLUG || 'general';
}

// Default: short warm-up then measure.
// LIGHT=1 → low concurrency for Docker PG with small /dev/shm (avoids shared-memory OOM).
// Full stages (50 VUs) can exhaust Compose postgres shm=64m under ListTopics COUNT/sort.
export function defaultStages() {
  if (__ENV.LIGHT === '1') {
    return [
      { duration: '5s', target: 2 }, // warm-up
      { duration: '10s', target: 5 }, // ramp
      { duration: '30s', target: 5 }, // measure
      { duration: '5s', target: 0 },
    ];
  }
  return [
    { duration: '15s', target: 10 }, // warm-up
    { duration: '30s', target: 50 }, // ramp
    { duration: '60s', target: 50 }, // measure
    { duration: '10s', target: 0 },
  ];
}

export function defaultThresholds(p99Ms) {
  return {
    http_req_failed: ['rate<0.05'],
    http_req_duration: [`p(99)<${p99Ms}`],
  };
}

export function okJson(res) {
  if (res.status !== 200) {
    return null;
  }
  try {
    return res.json();
  } catch (e) {
    return null;
  }
}
