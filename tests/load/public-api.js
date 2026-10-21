// k6 load test for public read endpoints. Run:
//   k6 run tests/load/public-api.js
// Not part of unit CI; use against staging only.
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

const BASE = __ENV.BASE_URL || 'https://so-called-spark.ru';

export default function () {
  for (const path of ['/', '/api/v1/pricing', '/api/v1/stats/places', '/api/v1/health/ready']) {
    const res = http.get(`${BASE}${path}`);
    check(res, { [`${path} status 200`]: (r) => r.status === 200 });
  }
  sleep(1);
}
