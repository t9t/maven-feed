#!/bin/bash

set -o errexit
set -o nounset

GIT_HASH=$(git rev-parse --short HEAD)
IS_DIRTY=$(git diff --quiet && echo 0 || echo 1)

SUFFIX=

if [[ ${IS_DIRTY} -eq 1 ]]; then
  echo Dirty workdir, using _d suffix. Git status:
  git status -s
  SUFFIX=_d
else
  echo Workdir is clean
fi

TAG_NAME=${GIT_HASH}${SUFFIX}
BUILD_TIME=$(TZ=UTC date -Iseconds)

docker build --no-cache --build-arg COMMIT_HASH="${TAG_NAME}" --build-arg BUILD_TIME="${BUILD_TIME}" -t maven-feed:${TAG_NAME} -t maven-feed:latest .
