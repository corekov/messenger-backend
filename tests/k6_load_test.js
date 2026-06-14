import ws from 'k6/ws';
import { check, sleep } from 'k6';
import http from 'k6/http';

export const options = {
  stages: [
    { duration: '30s', target: 200 },  // Ramp-up to 200 users over 30 seconds
    { duration: '1m', target: 1000 },  // Ramp-up to 1000 concurrent WebSocket connections (NFR_003)
    { duration: '2m', target: 1000 },  // Maintain 1000 users for 2 minutes
    { duration: '30s', target: 0 },    // Ramp-down to 0 users
  ],
  thresholds: {
    'ws_sessions': ['count>=1000'],    // Ensure we reach 1000 sessions
  },
};

const BASE_URL = 'http://localhost:8080/api/v1';
const WS_URL = 'ws://localhost:8080/api/v1/ws';

export default function () {
  // 1. Authenticate user to get JWT token
  // For a real load test, you might want to pre-create users or use a mocked auth endpoint.
  // We simulate a login request here.
  const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    username: `load_user_${__VU}`, // Use Virtual User ID as username
    password: 'password123',
    device_fp: `device_${__VU}`
  }), { headers: { 'Content-Type': 'application/json' } });

  // If login fails, we shouldn't attempt WS connection
  let authData = null;
  if (loginRes.status === 200) {
    authData = loginRes.json();
  } else {
    // If the user doesn't exist, register them first
    http.post(`${BASE_URL}/auth/register`, JSON.stringify({
      username: `load_user_${__VU}`,
      password: 'password123',
      device_name: `Device ${__VU}`,
      device_fp: `device_${__VU}`,
      platform: 'android'
    }), { headers: { 'Content-Type': 'application/json' } });
    
    // Try login again
    const retryRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
      username: `load_user_${__VU}`,
      password: 'password123',
      device_fp: `device_${__VU}`
    }), { headers: { 'Content-Type': 'application/json' } });

    if (retryRes.status === 200) {
        authData = retryRes.json();
    }
  }

  if (!authData || !authData.access_token) {
    sleep(1); // Prevent tight loops that exhaust ports
    return;
  }

  const token = authData.access_token;

  // 2. Connect to WebSocket
  const url = `${WS_URL}?token=${token}`;
  
  const res = ws.connect(url, function (socket) {
    socket.on('open', () => {
      // Simulate sending a "ping" or "status" update every 10 seconds
      socket.setInterval(function timeout() {
        socket.send(JSON.stringify({
          type: "status_update",
          payload: { status: "online" }
        }));
      }, 10000);
    });

    socket.on('message', (msg) => {
      // Message received
      check(msg, { 'is valid string': (m) => typeof m === 'string' });
    });

    socket.on('error', (e) => {
      if (e.error() != 'websocket: close sent') {
        console.error('An unexpected error occurred: ', e.error());
      }
    });

    socket.on('close', () => {
      // Connection closed
    });

    // Keep the connection open for a while
    sleep(180);
    socket.close();
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
