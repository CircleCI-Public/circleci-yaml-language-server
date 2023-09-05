#!/usr/bin/env bash
# Inspired by https://belief-driven-design.com/build-time-variables-in-go-51439b26ef9/


PACKAGE="github.com/CircleCI-Public/circleci-yaml-language-server/pkg/server/methods"

SCRIPT_PATH=$(cd $(dirname $0) && pwd)
VERSION=$(cd $SCRIPT_PATH && go run ./get_next_release.go)

BUILD_TIMESTAMP=$(date '+%Y-%m-%dT%H:%M:%S')

LDFLAGS=(
  "-X '${PACKAGE}.ServerVersion=${VERSION}'"
  "-X '${PACKAGE}.BuildTime=${BUILD_TIMESTAMP}'"
  "-X '${PACKAGE}.RollbarToken=${ROLLBAR_TOKEN}'"
)

echo -n "-ldflags=\"${LDFLAGS[*]} $SUFFIX\""
