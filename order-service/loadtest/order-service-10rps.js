import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    ten_rps: {
      executor: 'constant-arrival-rate',
      rate: 10,
      timeUnit: '1s',
      duration: __ENV.DURATION || '10m',
      preAllocatedVUs: 20,
      maxVUs: 100,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300', 'p(99)<500'],
    checks: ['rate>0.99'],
  },
};

function uuid() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = Math.floor(Math.random() * 16);
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export default function () {
  const idempotencyKey = `k6-${__VU}-${__ITER}-${Date.now()}`;
  const payload = JSON.stringify({
    customer_id: uuid(),
    currency: 'INR',
    items: [{ product_id: uuid(), quantity: 2, unit_price: 149900 }],
  });

  const response = http.post(`${__ENV.BASE_URL}/api/v1/orders`, payload, {
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
    },
    timeout: '3s',
  });

  check(response, {
    'created': (r) => r.status === 201,
    'has order id': (r) => {
      try { return Boolean(r.json('id')); } catch (_) { return false; }
    },
  });
}
