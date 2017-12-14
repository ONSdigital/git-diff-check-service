#!/bin/sh

set -e -x

# Shouldn't need to do this!? Go doesn't appear to be respecting the vendor/ path
# in the main source (although that may be due to how the code is getting loaded
# into the image)
# If you want to run the unit tests of the vendor'd libraries themselves then
# do a `cp -r` instead to leave them locally
echo "Moving app to gopath"

APP="git-diff-check-service"
WORKDIR=${GOPATH}/src/github.com/ONSdigital/${APP}


mkdir -p ${WORKDIR}
cp -r $PWD/${APP}/* ${WORKDIR}

echo "Executing tests"
cd ${WORKDIR}
ls
go test ./...