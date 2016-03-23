#!/bin/bash

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

CURDIR=`pwd`
GOIARDI_VERSION=`head -n 1 $CURDIR/../CHANGELOG | cut -f 1 -d ' '`
ITERATION=`cat $DIR/iteration`

gem install package_cloud
# if we're here, we're deploying. Unleash the tag
git tag "pkg-${GOIARDI_VERSION}-${ITERATION}"
git push --tags
git tag -d "pkg-${GOIARDI_VERSION}-${ITERATION}"

if [ -z ${PACKAGECLOUD_REPO} ] ; then
  echo "The environment variable PACKAGECLOUD_REPO must be set."
  exit 1
fi

# debian/raspbian
package_cloud push ${PACKAGECLOUD_REPO}/debian/wheezy ${DIR}/artifacts/goiardi-${GOIARDI_VERSION}-${ITERATION}_*.deb

# debian/jessie
package_cloud push ${PACKAGECLOUD_REPO}/debian/jessie ${DIR}/artifacts/goiardi-${GOIARDI_VERSION}-${ITERATION}jessie_*.deb

# ubuntu
package_cloud push ${PACKAGECLOUD_REPO}/ubuntu/trusty ${DIR}/artifacts/goiardi-${GOIARDI_VERSION}-${ITERATION}ubuntu_*.deb
