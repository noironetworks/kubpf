#!/bin/bash
if [ -z $1 ]
then
  DOCKER_BUILDKIT=1 docker build -f Dockerfile-build --target artifacts --output type=local,dest=. .
elif [ "$1" == "no-cache" ]
then
  DOCKER_BUILDKIT=1 docker build --no-cache -f Dockerfile-build --target artifacts --output type=local,dest=. .
else
  echo -e "Unrecognized argument $1.\nUsage: build.sh <no-cache>"
  exit
fi
docker build -f Dockerfile -t stalactite/statsagent:latest .
