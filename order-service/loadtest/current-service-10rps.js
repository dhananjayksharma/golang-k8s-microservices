import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    current_api: {
      executor: 'constant-arrival-rate',
      rate: Number(__ENV.RATE || 10),
      timeUnit: '1s',
      duration: __ENV.DURATION || '2m',
      preAllocatedVUs: 10,
      maxVUs: 50,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300'],
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
  const payload = JSON.stringify({
    customer_id: uuid(),
    idempotency_key: `k6-${__VU}-${__ITER}-${Date.now()}`,
    sku: __ENV.SKU || 'MOUSE-001',
    quantity: 2,
    unit_price: 149900,
    currency: 'INR',
    tax_amount: 0,
    shipping_amount: 0,
  });

  const response = http.post(`${__ENV.BASE_URL || 'http://localhost:8081'}/orders`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(response, {
    'status 201': (r) => r.status === 201,
    'standard response': (r) => r.json('kind') === 'standard',
    'has order id': (r) => Boolean(r.json('data.id')),
    'sku preserved': (r) => r.json('data.sku') === (__ENV.SKU || 'MOUSE-001'),
  });
}
