#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:?PROJECT_ID is required}"
REGION="${REGION:-asia-south1}"
REPOSITORY="${REPOSITORY:-notes-platform}"
SERVICE_NAME="${SERVICE_NAME:-notes-platform}"
IMAGE_NAME="${IMAGE_NAME:-web}"
IMAGE_TAG="${IMAGE_TAG:-$(git rev-parse --short HEAD)}"

IMAGE_URI="$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$IMAGE_NAME:$IMAGE_TAG"

echo "Using image: $IMAGE_URI"

echo "Building Docker image..."
docker build -t "$IMAGE_URI" .

echo "Pushing Docker image..."
docker push "$IMAGE_URI"

echo "Deploying Cloud Run service..."
gcloud run deploy "$SERVICE_NAME" \
  --image="$IMAGE_URI" \
  --region="$REGION"

echo "Deployment complete."

SERVICE_URL="$(gcloud run services describe "$SERVICE_NAME" \
  --region="$REGION" \
  --format='value(status.url)')"

echo "Service URL: $SERVICE_URL"