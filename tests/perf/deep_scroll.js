/**
 * M5 deep-scroll scenario: many list pages via keyset `after` vs deep OFFSET `page`.
 *
 * Prefer LIGHT=1 on Compose PG. Records p50/p99 across N continuation steps.
 *
 * Env:
 *   BASE_URL       default http://127.0.0.1:8081
 *   CATEGORY_SLUG  default general
 *   DEEP_STEPS     number of pages to walk (default 25)
 *   PER_PAGE       default 20
 *   MODE           cursor | page | both (default both)
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { api, defaultStages, defaultThresholds } from './lib.js';

const steps = Number(__ENV.DEEP_STEPS || 25);
const perPage = Number(__ENV.PER_PAGE || 20);
const category = __ENV.CATEGORY_SLUG || 'general';
const mode = (__ENV.MODE || 'both').toLowerCase();

export const options = {
  stages: defaultStages(),
  // Deep keyset should stay stable; page mode may degrade (documented residual).
  thresholds: defaultThresholds(200),
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

function walkCursor() {
  let after = '';
  let ok = true;
  for (let i = 0; i < steps; i++) {
    const q = after
      ? `/topics?categorySlug=${encodeURIComponent(category)}&perPage=${perPage}&after=${encodeURIComponent(after)}`
      : `/topics?categorySlug=${encodeURIComponent(category)}&perPage=${perPage}&page=1`;
    const res = http.get(api(q), { tags: { name: 'deep_scroll_cursor' } });
    const bodyOk = check(res, {
      'cursor status 200': (r) => r.status === 200,
      'cursor has items': (r) => {
        try {
          const body = r.json();
          return body && body.data && Array.isArray(body.data.items) && body.data.items.length > 0;
        } catch (e) {
          return false;
        }
      },
    });
    if (!bodyOk) {
      ok = false;
      break;
    }
    try {
      const data = res.json().data;
      if (!data.hasMore || !data.nextCursor) {
        break;
      }
      after = data.nextCursor;
    } catch (e) {
      ok = false;
      break;
    }
  }
  return ok;
}

function walkPage() {
  let ok = true;
  // Deep OFFSET: start near clamp region (page 50–min(steps,200))
  const start = Math.max(1, Math.min(50, 200 - steps));
  for (let i = 0; i < steps; i++) {
    const page = Math.min(200, start + i);
    const q = `/topics?categorySlug=${encodeURIComponent(category)}&perPage=${perPage}&page=${page}`;
    const res = http.get(api(q), { tags: { name: 'deep_scroll_page' } });
    const bodyOk = check(res, {
      'page status 200': (r) => r.status === 200,
    });
    if (!bodyOk) {
      ok = false;
      break;
    }
  }
  return ok;
}

export default function () {
  if (mode === 'cursor' || mode === 'both') {
    walkCursor();
  }
  if (mode === 'page' || mode === 'both') {
    walkPage();
  }
  sleep(0.05);
}
