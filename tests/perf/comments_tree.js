import http from 'k6/http';
import { check, sleep } from 'k6';
import { api, hotSlug, defaultStages, defaultThresholds, okJson } from './lib.js';

export const options = {
  stages: defaultStages(),
  thresholds: defaultThresholds(120), // M3: descendants capped (default 50/root) + ListComments cache
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export function setup() {
  const res = http.get(api(`/topics/by-slug/${encodeURIComponent(hotSlug())}`));
  const body = okJson(res);
  if (!body || !body.data || !body.data.id) {
    throw new Error(`setup: hot topic ${hotSlug()} not found (status ${res.status})`);
  }
  return { topicId: body.data.id };
}

export default function (data) {
  const id = data.topicId;
  const res = http.get(api(`/topics/${id}/comments?view=tree&page=1&perPage=20`), {
    tags: { name: 'comments_tree_p1' },
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
    // M0: record body size; M3 will bound descendants.
    'body under 20MiB': (r) => r.body && r.body.length < 20 * 1024 * 1024,
  });
  sleep(0.08);
}
