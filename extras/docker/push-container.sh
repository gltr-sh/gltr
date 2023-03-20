#! /usr/bin/env bash

CONTAINER_IMAGE="gltr/minimal-notebook"

echo
echo "Tagging for sean's registry..."
docker tag ${CONTAINER_IMAGE} gltr/minimal-notebook

echo
echo "Pushing to dockerhub..."
docker push gltr/minimal-notebook
