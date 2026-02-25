import http from 'k6/http';
import { check } from 'k6';

const WALLET_ID = '11111111-1111-1111-1111-111111111111';
const BASE_URL = 'http://localhost:8080';

export const options = {
  scenarios: {
    mixed_operations: {
      executor: 'constant-vus',
      vus: 100,
      duration: '10s'
    }
  },
  thresholds: {
    http_req_duration: ['p(99)<500'],
    http_req_failed: ['rate==0']
  }
};

const HEADERS = { 'Content-Type': 'application/json' };

export default function () {
    const rand = Math.random();
  
    let res;
    if (rand < 0.4) {
      res = http.post(
        BASE_URL + '/api/v1/wallet',
        JSON.stringify({ valletId: WALLET_ID, operationType: 'DEPOSIT', amount: 10 }),
        { headers: HEADERS }
      );
    } else if (rand < 0.8) {
      res = http.post(
        BASE_URL + '/api/v1/wallet',
        JSON.stringify({ valletId: WALLET_ID, operationType: 'WITHDRAW', amount: 1 }),
        { headers: HEADERS }
      );
    } else {
      res = http.get(BASE_URL + '/api/v1/wallets/' + WALLET_ID);
    }
  
    // логируем все не-200 и не-409 ответы
    if (res.status !== 200 && res.status !== 409 && res.status !== 0) {
      console.log('unexpected status: ' + res.status + ' body: ' + res.body);
    }
    if (res.status === 0) {
      console.log('connection failed: ' + res.error);
    }
  }