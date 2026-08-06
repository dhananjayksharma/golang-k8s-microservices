import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    ten_rps: {
      executor: 'constant-arrival-rate',
      rate: Number(__ENV.RATE || 10),
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

export function setup() {
  const baseURL = __ENV.BASE_URL || 'http://localhost:8081';
  const ready = http.get(`${baseURL}/health/ready`);
  check(ready, {
    'service ready': (r) => r.status === 200 && r.json('data.status') === 'ready',
  });
  return { baseURL };
}

export default function (state) {
  const sku = __ENV.SKU || 'MOUSE-001';
  const payload = JSON.stringify({
    customer_id: uuid(),
    idempotency_key: `k6-${__VU}-${__ITER}-${Date.now()}`,
    sku,
    quantity: 2,
    unit_price: 149900,
    currency: 'INR',
    tax_amount: 0,
    shipping_amount: 0,
  });

  const create = http.post(`${state.baseURL}/orders`, payload, {
    headers: { 'Content-Type': 'application/json' },
    timeout: '3s',
  });

  const created = check(create, {
    'created': (r) => r.status === 201,
    'standard response': (r) => r.json('kind') === 'standard',
    'has order id': (r) => Boolean(r.json('data.id')),
    'subtotal calculated': (r) => r.json('data.subtotal') === 299800,
    'total calculated': (r) => r.json('data.total_amount') === 299800,
  });

  if (!created) return;

  const orderID = create.json('data.id');
  sleep(0.1);
  const get = http.get(`${state.baseURL}/orders/${orderID}`, { timeout: '3s' });
  check(get, {
    'read succeeds': (r) => r.status === 200,
    'same order id': (r) => r.json('data.id') === orderID,
    'same sku': (r) => r.json('data.sku') === sku,
  });
}
