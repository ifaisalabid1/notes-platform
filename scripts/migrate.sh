#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:?PROJECT_ID is required}"
REGION="${REGION:-asia-southeast1}"
REPOSITORY="${REPOSITORY:-notes-platform}"
JOB_NAME="${JOB_NAME:-notes-platform-migrate}"
IMAGE_NAME="${IMAGE_NAME:-web}"
IMAGE_TAG="${IMAGE_TAG:-$(git rev-parse --short HEAD)}"

IMAGE_URI="$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$IMAGE_NAME:$IMAGE_TAG"

echo "Using migration image: $IMAGE_URI"

echo "Updating Cloud Run migration job..."
gcloud run jobs update "$JOB_NAME" \
  --image="$IMAGE_URI" \
  --region="$REGION" \
  --command="/app/migrate" \
  --args="up" \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest"

echo "Executing migration job..."
gcloud run jobs execute "$JOB_NAME" \
  --region="$REGION" \
  --wait

echo "Migration complete."