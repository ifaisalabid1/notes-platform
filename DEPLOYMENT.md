# Notes Platform Deployment Guide

This project deploys to Google Cloud Run, uses Neon PostgreSQL, Cloudflare R2 for private file storage, and a Cloudflare Worker for secure file streaming.

## Required tools

Install and configure:

- Docker
- Google Cloud CLI
- pnpm
- Wrangler, for Cloudflare Worker deployment

## Production services

- Google Cloud Run: Go web app
- Google Artifact Registry: Docker image storage
- Google Secret Manager: production secrets
- Cloud Run Job: database migrations
- Neon: PostgreSQL
- Cloudflare R2: private file storage
- Cloudflare Worker: private file proxy

## Environment variables

Production Cloud Run service needs:

```env
APP_ENV=production
APP_HOST=0.0.0.0
APP_BASE_URL=https://your-domain.com

R2_ACCOUNT_ID=your-cloudflare-account-id
R2_BUCKET_NAME=your-private-r2-bucket-name

BRAND_NAME=Your Brand Name
BRAND_URL=https://your-domain.com
WATERMARK_TEXT=

FILE_PROXY_BASE_URL=https://files.your-domain.com
FILE_PROXY_URL_TTL_SECONDS=300
```
