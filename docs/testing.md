# Unified Testing Guide

This single guide covers testing for the current project scope (Phase 4 through Phase 8).

## 1) Quick commands

From repo root:

```bash
cd labra-backend
mkdir -p .gocache
GOCACHE=$(pwd)/.gocache go test ./... -count=1
rm -rf .gocache
```

```bash
cd ../labra-frontend
npm run check
```

If `npm` is missing in your shell, load your Node environment (for example via your `nvm` profile) and rerun.

## 2) Test structure

Backend test files live in `labra-backend/internal/api/handlers/`:

- `phase4_webhook_test.go`: webhook signature, routing, dedupe, metadata history.
- `phase6_env_health_test.go`: env var CRUD/masking and health metrics/summary.
- `phase5_phase7_rollback_observability_test.go`: releases, rollback timeline, observability APIs.
- `phase8_queue_retries_test.go`: queue worker, retries/backoff, queue status, idempotency.
- `core_smoke_test.go`: end-to-end core flow smoke test.

These are in-memory SQLite handler/integration-style tests, so they are fast and realistic.

## 3) What “core smoke” proves

`core_smoke_test.go` validates the essential path:

1. Manual deploys are queued and processed by worker.
2. Queue status endpoint responds for deployments.
3. Release pointer updates after successful deploys.
4. Rollback request is queued and processed.
5. Health and observability endpoints return expected shape and data.

If this test fails, the platform’s primary user workflow is likely broken.

## 4) Manual API smoke checks

Start backend:

```bash
cd labra-backend
make run
```

Assume `X-User-ID: 1` and app ID `1`.

### Deploy and inspect queue

```bash
curl -s -X POST http://localhost:8080/v1/apps/1/deploy -H 'X-User-ID: 1'
curl -s http://localhost:8080/v1/deploys/1 -H 'X-User-ID: 1'
curl -s http://localhost:8080/v1/deploys/1/queue -H 'X-User-ID: 1'
```

### Env vars and health

```bash
curl -s -X POST http://localhost:8080/v1/apps/1/env-vars \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"key":"API_TOKEN","value":"secret-value","is_secret":true}'

curl -s http://localhost:8080/v1/apps/1/env-vars -H 'X-User-ID: 1'
curl -s http://localhost:8080/v1/apps/1/health -H 'X-User-ID: 1'
```

### Releases, rollback, observability

```bash
curl -s http://localhost:8080/v1/apps/1/releases -H 'X-User-ID: 1'

curl -s -X POST http://localhost:8080/v1/apps/1/rollback \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"reason":"manual verification"}'

curl -s http://localhost:8080/v1/apps/1/rollbacks -H 'X-User-ID: 1'
curl -s http://localhost:8080/v1/apps/1/observability -H 'X-User-ID: 1'
curl -s "http://localhost:8080/v1/apps/1/observability/log-query?q=build&source=local" -H 'X-User-ID: 1'
```

## 5) Retry simulation (Phase 8)

Use env vars to simulate failure modes:

- Transient retry path:
  - key: `LABRA_FORCE_TRANSIENT_FAILURES`
  - value: `1`
- Permanent failure path:
  - key: `LABRA_FORCE_PERMANENT_FAILURE`
  - value: `true`

Then trigger deploy and inspect `/v1/deploys/:id/queue` and `/v1/deploys/:id`.

## 6) CI/CD behavior

CI workflow: `.github/workflows/ci.yml`

Automatically runs on PRs and selected branches:

1. Backend tests: `go test ./...`
2. Frontend checks: `npm ci` + `npm run check`

This is your regression gate before merge.
