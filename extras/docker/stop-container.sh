#! /usr/bin/env bash

CONTAINER_NAME="jn"
CONTAINER_IMAGE="gltr/minimal-notebook"

echo
echo "Stopping container (if running)..."
docker stop ${CONTAINER_NAME}
