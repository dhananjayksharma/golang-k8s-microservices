import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    current_api: {
      executor: 'constant-arrival-rate',
      rate: 10,
      timeUnit: '1s',
      duration: __ENV.DURATION || '2m',
      preAllocatedVUs: 10,
      maxVUs: 50,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300'],
  },
};

export default function () {
  const payload = JSON.stringify({
    quantity: 2,
    price: 1499.0,
    date: new Date().toISOString(),
  });
  const response = http.post(`${__ENV.BASE_URL}/orders/`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(response, { 'status 200': (r) => r.status === 200 });
}
