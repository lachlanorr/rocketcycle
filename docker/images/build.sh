#!/bin/bash

# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.

set -e

function help_and_exit
{
    echo "build.sh [--official] <tag>"
    exit
}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

OFFICIAL=false
HELP=false

for var in "$@"
do
    if [ "$var" = "--help" ] ; then
        help_and_exit
    elif [ "$var" = "--official" ] ; then
        OFFICIAL=true
    elif [ -z "$IMAGETAG" ] ; then
        IMAGETAG="$var"
    else
        ARGS="$ARGS $var"
    fi
done

if [ -z "$IMAGETAG" ] ; then
    help_and_exit
fi

IMAGETAG_TOKS=(${IMAGETAG//:/ })
IMAGE=${IMAGETAG_TOKS[0]}
TAG=${IMAGETAG_TOKS[1]:-latest}

SRCDIR=${DIR}/${IMAGE}

echo "BUILDDIR=$BUILDDIR"
echo "OFFICIAL=$OFFICIAL"
echo "IMAGE=$IMAGE"
echo "TAG=$TAG"
echo "ARGS=$ARGS"

BUILDDIR=${DIR}/__build_${IMAGE}
rm -rf $BUILDDIR
mkdir $BUILDDIR

cp -r $SRCDIR/* $BUILDDIR

cd $BUILDDIR

if test -f "prebuild.sh" ; then
    . prebuild.sh
fi

echo "KAFKA_TGZ=${KAFKA_TGZ}"

echo "docker build . -t ${TAG} ${ARGS}"
cat Dockerfile | envsubst > Dockerfile_subst
rm -rf Dockerfile
mv Dockerfile_subst Dockerfile
rm -rf Dockerfile_subst

docker build . -t rocketcycle-${IMAGETAG} ${ARGS}
