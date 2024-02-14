#!/usr/bin/env bash
# Inspired by https://belief-driven-design.com/build-time-variables-in-go-51439b26ef9/

if [ ! -f ~/version ];
then
	>&2 echo "No version file defined. Returning invalid build argument"
	echo "invalid-argument"
	exit 1
fi

PACKAGE="github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"

VERSION=$(cat ~/version)

LDFLAGS=(
  "-X '${PACKAGE}.ServerVersion=${VERSION}'"
)

echo -n "-ldflags=\"${LDFLAGS[*]} $SUFFIX\""
