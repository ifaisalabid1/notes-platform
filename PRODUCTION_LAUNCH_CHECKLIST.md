# Rising Star Production Launch Checklist

Use this checklist before making the site public.

## 1. Code readiness

- [ ] All changes are committed.
- [ ] Docker image builds successfully.
- [ ] App runs locally with Docker.
- [ ] `pnpm run css:build` works.
- [ ] `go run ./cmd/web` works locally.
- [ ] `go run ./cmd/migrate status` works locally.
- [ ] No temporary test routes are enabled.
- [ ] No debug logs expose secrets.
- [ ] No `.env` file is committed.
- [ ] `.dockerignore` excludes local-only files.

## 2. Production environment variables

Cloud Run service should have:

- [ ] `APP_ENV=production`
- [ ] `APP_HOST=0.0.0.0`
- [ ] `APP_BASE_URL=https://your-domain.com`
- [ ] `R2_ACCOUNT_ID`
- [ ] `R2_BUCKET_NAME`
- [ ] `BRAND_NAME`
- [ ] `BRAND_URL=https://your-domain.com`
- [ ] `WATERMARK_TEXT`
- [ ] `FILE_PROXY_BASE_URL=https://files.your-domain.com`
- [ ] `FILE_PROXY_URL_TTL_SECONDS=300`

Do not manually set `PORT` unless needed locally. Cloud Run injects it.

## 3. Production secrets

Google Secret Manager should contain:

- [ ] `DATABASE_URL`
- [ ] `SESSION_SECRET`
- [ ] `R2_ACCESS_KEY_ID`
- [ ] `R2_SECRET_ACCESS_KEY`
- [ ] `FILE_PROXY_SECRET`

Temporary bootstrap secrets:

- [ ] `OWNER_NAME`
- [ ] `OWNER_EMAIL`
- [ ] `OWNER_PASSWORD`

After owner bootstrap:

- [ ] Owner password is changed from `/admin/account/password`.
- [ ] `OWNER_PASSWORD` secret is deleted.

## 4. Database

- [ ] Neon production database exists.
- [ ] Production database URL is correct.
- [ ] Connection pooling choice is confirmed.
- [ ] Migrations run through Cloud Run Job, not app startup.
- [ ] Latest migration version is applied.
- [ ] `admin_audit_logs` table exists.
- [ ] `archived_at` exists on notes.
- [ ] Search indexes exist.
- [ ] Session table exists and works.
- [ ] `MAINTENANCE_MODE=false`

Check migration job logs:

```bash
gcloud logging read \
  'resource.type="cloud_run_job" AND resource.labels.job_name="notes-platform-migrate"' \
  --limit=100
```
