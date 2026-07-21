import http from 'k6/http';
import { check, sleep } from 'k6';
import { api, categorySlug, defaultStages, defaultThresholds } from './lib.js';

export const options = {
  stages: defaultStages(),
  thresholds: defaultThresholds(200),
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export default function () {
  const slug = categorySlug();
  const res = http.get(api(`/topics?page=1&perPage=20&categorySlug=${encodeURIComponent(slug)}`), {
    tags: { name: 'category_topics_p1' },
  });
  check(res, {
    'status 200': (r) => r.status === 200,
    'has items': (r) => {
      try {
        const body = r.json();
        return body && body.data && Array.isArray(body.data.items);
      } catch (e) {
        return false;
      }
    },
  });
  sleep(0.05);
}
