import http from 'k6/http';
import { check, sleep } from 'k6';
import { api, categorySlug, defaultStages, defaultThresholds } from './lib.js';

// 90% read / 10% write. Write requires AUTH_COOKIE (session cookie header value).
// Without AUTH_COOKIE, write branch is skipped and counted as read-only mixed traffic.

export const options = {
  stages:
    __ENV.LIGHT === '1'
      ? [
          { duration: '5s', target: 2 },
          { duration: '20s', target: 5 },
          { duration: '5s', target: 0 },
        ]
      : [
          { duration: '10s', target: 10 },
          { duration: '20s', target: 30 },
          { duration: '40s', target: 30 },
          { duration: '10s', target: 0 },
        ],
  thresholds: defaultThresholds(500),
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export default function () {
  const roll = Math.random();
  const auth = __ENV.AUTH_COOKIE || '';

  if (roll < 0.1 && auth) {
    const payload = JSON.stringify({
      categorySlug: categorySlug(),
      title: `k6 mixed write ${Date.now()} ${Math.random()}`,
      content: {
        rawContent: 'k6 mixed-read-write probe body',
        sourceFormat: 'markdown',
        editorType: 'markdown',
      },
    });
    const res = http.post(api('/topics'), payload, {
      headers: {
        'Content-Type': 'application/json',
        Cookie: auth,
      },
      tags: { name: 'mixed_write_topic' },
    });
    check(res, {
      'write status 2xx or 4xx': (r) => r.status >= 200 && r.status < 500,
    });
  } else {
    const res = http.get(api('/topics?page=1&perPage=20'), {
      tags: { name: 'mixed_read_home' },
    });
    check(res, {
      'read status 200': (r) => r.status === 200,
    });
  }
  sleep(0.05);
}
