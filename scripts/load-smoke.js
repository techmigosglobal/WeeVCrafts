import http from 'k6/http';
import { check, sleep } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '60s', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
  },
};

export default function () {
  const browse = http.get(`${baseURL}/api/v1/products`);
  check(browse, { 'browse returns JSON success': (response) => response.status === 200 && response.headers['Content-Type'].includes('application/json') });

  const search = http.get(`${baseURL}/api/v1/search?q=soap&limit=24`);
  check(search, { 'search returns success': (response) => response.status === 200 });

  const health = http.get(`${baseURL}/health/live`);
  check(health, { 'live probe remains available': (response) => response.status === 200 });
  sleep(1);
}
