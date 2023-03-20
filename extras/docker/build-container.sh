#! /usr/bin/env bash

CURRENT_DIR=$(pwd)
TMPDIR=/tmp/gltr-build
NB_UID=1000
NB_USER=gltr
OWNER=gltr
PLATFORM="linux/arm64"
#PLATFORM="linux/amd64,linux/arm64"

echo "Setting up QEMU emulators.." 
docker run -it --rm --privileged tonistiigi/binfmt --install all

echo
echo "Making temporary directory..."
mkdir -p $TMPDIR
cd $TMPDIR

echo
echo "Setup buildx builder"
docker buildx create --use --name gltr-builder

echo
echo "Cloning jupyter git repo..."
git clone https://github.com/jupyter/docker-stacks.git

echo
echo "Building foundation-notebook ..."
cd $TMPDIR/docker-stacks/docker-stacks-foundation
docker buildx build \
  --platform $PLATFORM \
  --build-arg NB_UID=$NB_UID \
  --build-arg NB_USER=$NB_USER \
  --cache-from gltr/docker-stacks-foundation:cache \
  --cache-to gltr/docker-stacks-foundation:cache \
  --push \
  -t gltr/docker-stacks-foundation .

echo
echo "Building base-notebook ..."
cd $TMPDIR/docker-stacks//base-notebook
docker buildx build \
  --platform $PLATFORM \
  --build-arg NB_UID=$NB_UID \
  --build-arg NB_USER=$NB_USER \
  --build-arg OWNER=$OWNER \
  --cache-from gltr/base-notebook:cache \
  --cache-to gltr/base-notebook:cache \
  --push \
  -t gltr/base-notebook .

echo
echo "Removing temporary directory..."
cd $CURRENT_DIR
rm -fr $TMPDIR

# assume gltr has been already built...
cp ../../gltr .

echo
echo "Building minimal notebook ..."
docker buildx build \
  --platform $PLATFORM \
  --cache-from gltr/minimal-notebook:cache \
  --cache-to gltr/minimal-notebook:cache \
  --push \
  -t gltr/minimal-notebook .

rm ./gltr
