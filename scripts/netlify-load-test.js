import http from 'k6/http';
import { check, sleep } from 'k6';

const baseURL = (__ENV.BASE_URL || '').replace(/\/$/, '');
const email = __ENV.CUSTOMER_EMAIL || '';
const password = __ENV.CUSTOMER_PASSWORD || '';
let signedIn = false;

export const options = {
  scenarios: {
    marketplace: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 20 },
        { duration: '60s', target: 20 },
        { duration: '15s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<2500'],
  },
};

export default function () {
  if (!baseURL || !email || !password) {
    throw new Error('Set BASE_URL, CUSTOMER_EMAIL, and CUSTOMER_PASSWORD for a staging customer account.');
  }

  if (!signedIn) {
    const loginPage = http.get(`${baseURL}/login`, { tags: { workflow: 'login-page' } });
    const tokenMatch = loginPage.body.match(/name="csrf_token" value="([^"]+)"/);
    const pageOK = check(loginPage, { 'login page renders': (response) => response.status === 200 });
    const tokenOK = check(tokenMatch, { 'login CSRF token is present': (match) => match !== null });
    if (!pageOK || !tokenOK) return;

    const login = http.post(`${baseURL}/login`, {
      email,
      password,
      csrf_token: tokenMatch[1],
    }, {
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      redirects: 0,
      tags: { workflow: 'login' },
    });
    signedIn = check(login, { 'customer login succeeds': (response) => response.status === 303 });
    if (!signedIn) return;
  }

  const routes = [
    ['home', '/'],
    ['catalogue', '/products'],
    ['search', '/search?q=handmade'],
    ['catalogue-api', '/api/v1/products'],
    ['account', '/account/sessions'],
    ['cart', '/cart'],
    ['wishlist', '/wishlist'],
    ['orders', '/orders'],
    ['readiness', '/health/ready'],
  ];
  for (const [workflow, path] of routes) {
    const response = http.get(`${baseURL}${path}`, { tags: { workflow } });
    check(response, { [`${workflow} remains available`]: (result) => result.status === 200 });
  }
  sleep(1);
}
