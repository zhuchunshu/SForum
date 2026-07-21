import http from 'k6/http';
import { check, sleep } from 'k6';
import { api, hotSlug, defaultThresholds } from './lib.js';

// Flood public detail GET (by-slug). Until M2/Iteration A, each hit may UPDATE topics.view_count
// in PG — baseline should record that, not implement Redis view counting here.

export const options = {
  stages:
    __ENV.LIGHT === '1'
      ? [
          { duration: '5s', target: 5 },
          { duration: '20s', target: 10 },
          { duration: '5s', target: 0 },
        ]
      : [
          { duration: '10s', target: 50 },
          { duration: '20s', target: 200 },
          { duration: '40s', target: 200 },
          { duration: '10s', target: 0 },
        ],
  thresholds: defaultThresholds(300),
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export default function () {
  const res = http.get(api(`/topics/by-slug/${encodeURIComponent(hotSlug())}`), {
    tags: { name: 'view_flood_detail' },
  });
  check(res, {
    'status 200': (r) => r.status === 200,
  });
  sleep(0.01);
}
