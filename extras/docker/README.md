# Docker

This directory contains content required to build the docker container which is
(currently) used by `gltr`. The docker image is based on the jupyter
base-notebook container image and adds `s6` process management and an `sshd`
which runs in the container. The content in the `resources` directory is what
is required for `s6` process management.

We don't have a complete workflow here yet, primarily because we don't have a
proper home for the resulting containers; however, the instructions to build
the containers are here and how I push them to my own dockerhub repo are
provided; the next step is to address the container registry issue.

Note that the build process requires `docker buildx` - for Ubuntu this
can be downloaded from here: https://github.com/docker/buildx#manual-download

Currently, the image is available at

```
gltr/minimal-notebook
```

on `dockerhub`.
