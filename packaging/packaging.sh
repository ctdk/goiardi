#!/bin/bash

# For now, this will build debs for ubuntu 14.04 and debian wheezy on amd64.
# Requires gox and fpm to be installed.

# make more easily specified later
GOIARDI_VERSION="0.10.0"
ITERATION=`date +%s`

mkdir -p artifacts
git tag "pkg-${GOIARDI_VERSION}-${ITERATION}"
git push --tags

cd ..
gox -osarch="linux/amd64" -output="{{.Dir}}-$GOIARDI_VERSION-{{.OS}}-{{.Arch}}"

cd packaging/ubuntu/trusty
mkdir -p fs/usr/bin
mkdir -p fs/var/lib/goiardi/lfs
cp ../../../goiardi-$GOIARDI_VERSION-linux-amd64 fs/usr/bin/goiardi

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION --iteration ${ITERATION}ubuntu -C ./fs/ -p ../../artifacts/goiardi-VERSION-ITERATION_ARCH.deb -a amd64 --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cd ../../debian/wheezy
mkdir -p fs/usr/bin
mkdir -p fs/var/lib/goiardi/lfs
cp ../../../goiardi-$GOIARDI_VERSION-linux-amd64 fs/usr/bin/goiardi
fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION --iteration $ITERATION -C ./fs/ -p ../../artifacts/goiardi-VERSION-ITERATION_ARCH.deb -a amd64 --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cd ../../..

GOARM=6 gox -osarch="linux/arm" -output="{{.Dir}}-$GOIARDI_VERSION-{{.OS}}-{{.Arch}}6"
GOARM=7 gox -osarch="linux/arm" -output="{{.Dir}}-$GOIARDI_VERSION-{{.OS}}-{{.Arch}}7"

cd packaging/debian/wheezy
cp ../../../goiardi-$GOIARDI_VERSION-linux-arm6 fs/usr/bin/goiardi
fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION --iteration $ITERATION -C ./fs/ -p ../../artifacts/goiardi-VERSION-ITERATION_ARCH.deb -a armel --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cp ../../../goiardi-$GOIARDI_VERSION-linux-arm7 fs/usr/bin/goiardi
fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION --iteration $ITERATION -C ./fs/ -p ../../artifacts/goiardi-VERSION-ITERATION_ARCH.deb -a armhf --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cd ../..
