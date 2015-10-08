#!/bin/bash

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

GOIARDI_VERSION="0.10.1"
ITERATION=`date +%s`

gem install package_cloud
# if we're here, we're deploying. Unleash the tag
git push --tags

if [ -z ${PACKAGECLOUD_REPO} ] ; then
  echo "The environment variable PACKAGECLOUD_REPO must be set."
  exit 1
fi

# debian/raspbian
package_cloud push ${PACKAGECLOUD_REPO}/debian/wheezy ${DIR}/artifacts/goiardi-${GOIARDI_VERSION}-${ITERATION}_*.deb

# ubuntu
package_cloud push ${PACKAGECLOUD_REPO}/ubuntu/trusty ${DIR}/artifacts/goiardi-${GOIARDI_VERSION}-${ITERATION}ubuntu_*.deb
