import http from 'k6/http';
import { check, sleep } from 'k6';
import { api, hotSlug, defaultStages, defaultThresholds } from './lib.js';

export const options = {
  stages: defaultStages(),
  thresholds: defaultThresholds(150),
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export default function () {
  const slug = hotSlug();
  const res = http.get(api(`/topics/by-slug/${encodeURIComponent(slug)}`), {
    tags: { name: 'topic_by_slug' },
  });
  check(res, {
    'status 200': (r) => r.status === 200,
    'has topic id': (r) => {
      try {
        const body = r.json();
        return body && body.data && body.data.id;
      } catch (e) {
        return false;
      }
    },
  });
  sleep(0.03);
}
