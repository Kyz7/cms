#!/bin/bash

# Script untuk build Docker image CMS

set -e

IMAGE_NAME="cms-integrasi"
IMAGE_TAG="${1:-latest}"

echo "🐳 Building Docker image: ${IMAGE_NAME}:${IMAGE_TAG}"

# Build Docker image
docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .

echo "Build completed successfully!"
echo ""
echo "Image: ${IMAGE_NAME}:${IMAGE_TAG}"
echo ""
echo "To run the container:"
echo "   docker run -p 8080:8080 --env-file .env ${IMAGE_NAME}:${IMAGE_TAG}"
echo ""
echo "To test locally:"
echo "   docker run -p 8080:8080 \\"
echo "     -e DB_HOST=localhost \\"
echo "     -e DB_NAME=starpi \\"
echo "     -e DB_USER=postgres \\"
echo "     -e DB_PASSWORD=your_password \\"
echo "     -e JWT_SECRET=your_jwt_secret_minimum_32_characters_long \\"
echo "     ${IMAGE_NAME}:${IMAGE_TAG}"

