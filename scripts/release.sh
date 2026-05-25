#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:?PROJECT_ID is required}"
REGION="${REGION:-asia-south1}"
REPOSITORY="${REPOSITORY:-notes-platform}"
SERVICE_NAME="${SERVICE_NAME:-notes-platform}"
JOB_NAME="${JOB_NAME:-notes-platform-migrate}"
IMAGE_NAME="${IMAGE_NAME:-web}"
IMAGE_TAG="${IMAGE_TAG:-$(git rev-parse --short HEAD)}"

IMAGE_URI="$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$IMAGE_NAME:$IMAGE_TAG"

echo "Using release image: $IMAGE_URI"

echo "Building Docker image..."
docker build -t "$IMAGE_URI" .

echo "Pushing Docker image..."
docker push "$IMAGE_URI"

echo "Updating migration job..."
gcloud run jobs update "$JOB_NAME" \
  --image="$IMAGE_URI" \
  --region="$REGION" \
  --command="/app/migrate" \
  --args="up" \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest"

echo "Running migrations..."
gcloud run jobs execute "$JOB_NAME" \
  --region="$REGION" \
  --wait

echo "Deploying service..."
gcloud run deploy "$SERVICE_NAME" \
  --image="$IMAGE_URI" \
  --region="$REGION"

SERVICE_URL="$(gcloud run services describe "$SERVICE_NAME" \
  --region="$REGION" \
  --format='value(status.url)')"

echo "Release complete."
echo "Service URL: $SERVICE_URL"