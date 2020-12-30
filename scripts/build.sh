#!/bin/bash
DOCKER_BUILDKIT=1 docker build -f Dockerfile-build --target artifacts --output type=local,dest=. .
