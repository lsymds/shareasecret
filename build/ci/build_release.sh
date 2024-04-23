#!/bin/bash

print_help()
{
  echo "usage: ./build/ci/build_release.sh operating_system architecture version archive[true/false]"
  exit 1
}

if [[ $1 = "" ]] || [[ $2 = "" ]] || [[ $3 = "" ]]; then
  print_help
  exit 1
fi

export GOOS=$1
export GOARCH=$2

VERSION=$3
FILE_NAME="shareasecret-$GOOS-$GOARCH-$VERSION"

go build -o "./build/tmp/$FILE_NAME" -ldflags "-X main.version=$VERSION" .

if [[ $4 != "false" ]]; then
  bzip2 "./build/tmp/$FILE_NAME"
fi
