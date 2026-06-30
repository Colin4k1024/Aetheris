// k6 load test for Aetheris API
// Usage:
//   k6 run --env BASE_URL=http://localhost:8080 benchmarks/k6/load-test.js
//   k6 run --env BASE_URL=http://localhost:8080 --env VUS=50 --env DURATION=2m benchmarks/k6/load-test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const jobCreationTrend = new Trend('job_creation_duration');
const jobPollTrend = new Trend('job_poll_duration');
const jobsCreated = new Counter('jobs_created');
const jobsCompleted = new Counter('jobs_completed');
const jobsFailed = new Counter('jobs_failed');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const VUS = parseInt(__ENV.VUS) || 10;
const DURATION = __ENV.DURATION || '1m';

export const options = {
  stages: [
    { duration: '10s', target: VUS },       // Ramp up
    { duration: DURATION, target: VUS },     // Steady state
    { duration: '10s', target: 0 },          // Ramp down
  ],
  thresholds: {
    errors: ['rate<0.05'],                   // Error rate < 5%
    job_creation_duration: ['p(95)<500'],    // P95 job creation < 500ms
    job_poll_duration: ['p(95)<200'],        // P95 job poll < 200ms
  },
};

const AGENT_ID = __ENV.AGENT_ID || 'conversation';

function createJob() {
  const idempotencyKey = `load-test-${__VU}-${__ITER}-${Date.now()}`;
  const payload = JSON.stringify({
    message: `Load test job ${idempotencyKey}`,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
    },
    timeout: '10s',
  };

  const start = Date.now();
  const res = http.post(
    `${BASE_URL}/api/agents/${AGENT_ID}/message`,
    payload,
    params
  );
  const duration = Date.now() - start;

  jobCreationTrend.add(duration);

  const success = check(res, {
    'job creation status is 200 or 201': (r) => r.status === 200 || r.status === 201 || r.status === 202,
    'job creation has job_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return !!(body.job_id || body.id || (body.runtime_submission && body.runtime_submission.job_id));
      } catch {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);
    return null;
  }

  errorRate.add(0);
  jobsCreated.add(1);

  try {
    const body = JSON.parse(res.body);
    return body.job_id || body.id || (body.runtime_submission && body.runtime_submission.job_id);
  } catch {
    return null;
  }
}

function pollJob(jobId) {
  if (!jobId) return;

  const start = Date.now();
  const res = http.get(`${BASE_URL}/api/jobs/${jobId}`, {
    timeout: '5s',
  });
  const duration = Date.now() - start;

  jobPollTrend.add(duration);

  const success = check(res, {
    'job poll status is 200': (r) => r.status === 200,
  });

  if (!success) {
    errorRate.add(1);
    return;
  }

  errorRate.add(0);

  try {
    const body = JSON.parse(res.body);
    if (body.status === 'completed') {
      jobsCompleted.add(1);
    } else if (body.status === 'failed') {
      jobsFailed.add(1);
    }
  } catch {
    // Ignore parse errors
  }
}

function healthCheck() {
  const res = http.get(`${BASE_URL}/api/health`, { timeout: '5s' });
  check(res, {
    'health check status is 200': (r) => r.status === 200,
  });
}

export default function () {
  // 1. Health check (10% of iterations)
  if (Math.random() < 0.1) {
    healthCheck();
    sleep(0.5);
    return;
  }

  // 2. Create a job
  const jobId = createJob();

  // 3. Poll the job status
  if (jobId) {
    sleep(0.5);
    pollJob(jobId);
  }

  sleep(0.2);
}

export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    config: {
      vus: VUS,
      duration: DURATION,
      base_url: BASE_URL,
    },
    metrics: {
      jobs_created: data.metrics.jobs_created?.values?.count || 0,
      jobs_completed: data.metrics.jobs_completed?.values?.count || 0,
      jobs_failed: data.metrics.jobs_failed?.values?.count || 0,
      error_rate: data.metrics.errors?.values?.rate || 0,
      job_creation_p50: data.metrics.job_creation_duration?.values?.['p(50)'] || 0,
      job_creation_p95: data.metrics.job_creation_duration?.values?.['p(95)'] || 0,
      job_creation_p99: data.metrics.job_creation_duration?.values?.['p(99)'] || 0,
      job_poll_p50: data.metrics.job_poll_duration?.values?.['p(50)'] || 0,
      job_poll_p95: data.metrics.job_poll_duration?.values?.['p(95)'] || 0,
      http_req_duration_p95: data.metrics.http_req_duration?.values?.['p(95)'] || 0,
      http_req_duration_p99: data.metrics.http_req_duration?.values?.['p(99)'] || 0,
      http_reqs: data.metrics.http_reqs?.values?.count || 0,
    },
  };

  // Print summary to stdout
  console.log('\n=== Aetheris Load Test Summary ===');
  console.log(`VUs: ${summary.config.vus}, Duration: ${summary.config.duration}`);
  console.log(`Jobs Created: ${summary.metrics.jobs_created}`);
  console.log(`Jobs Completed: ${summary.metrics.jobs_completed}`);
  console.log(`Jobs Failed: ${summary.metrics.jobs_failed}`);
  console.log(`Error Rate: ${(summary.metrics.error_rate * 100).toFixed(2)}%`);
  console.log(`Job Creation P50: ${summary.metrics.job_creation_p50.toFixed(0)}ms`);
  console.log(`Job Creation P95: ${summary.metrics.job_creation_p95.toFixed(0)}ms`);
  console.log(`Job Creation P99: ${summary.metrics.job_creation_p99.toFixed(0)}ms`);
  console.log(`Job Poll P95: ${summary.metrics.job_poll_p95.toFixed(0)}ms`);
  console.log(`HTTP P95: ${summary.metrics.http_req_duration_p95.toFixed(0)}ms`);
  console.log(`HTTP P99: ${summary.metrics.http_req_duration_p99.toFixed(0)}ms`);
  console.log(`Total HTTP Requests: ${summary.metrics.http_reqs}`);
  console.log('================================\n');

  // Write JSON report
  const reportPath = 'benchmarks/reports/k6-summary.json';
  return {
    [reportPath]: JSON.stringify(summary, null, 2),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data) {
  return ''; // k6 default summary is printed automatically
}
