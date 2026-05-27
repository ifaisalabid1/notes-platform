# Rising Star Backup and Recovery Guide

This guide explains how to recover the Rising Star production system if something goes wrong.

The platform uses:

- Google Cloud Run for the Go web app
- Google Artifact Registry for Docker images
- Google Secret Manager for secrets
- Cloud Run Jobs for migrations and owner bootstrap
- Neon PostgreSQL for database
- Cloudflare R2 for private file storage
- Cloudflare Worker for private file streaming

## Recovery priorities

In an emergency, recover in this order:

1. Database
2. Uploaded files in R2
3. Cloud Run service
4. Cloudflare Worker
5. Secrets and environment variables

The database and R2 objects are the most important because they contain your actual content and file references.

---

## Database backup and recovery

The database stores:

- admins
- classes
- semesters
- subjects
- units
- chapters
- notes metadata
- R2 storage keys
- session data
- migration history

### Neon backups

Use Neon’s built-in backup, restore, branching, and point-in-time recovery features from the Neon dashboard.

Before major production changes:

- Create a Neon branch or restore point if available.
- Confirm the current production database connection string.
- Confirm migrations have been tested locally.

### Manual PostgreSQL backup

You can create a logical backup with `pg_dump`.

```bash
pg_dump "$DATABASE_URL" \
  --format=custom \
  --file=notes-platform-backup.dump
```
