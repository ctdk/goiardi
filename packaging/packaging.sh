#!/bin/bash

# For now, this will build debs for ubuntu 14.04 and debian wheezy on amd64.
# Requires gox and fpm to be installed.

# make more easily specified later
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

for VAR in trusty wheezy jessie; do
	mkdir -p artifacts/$VAR
done

CURDIR=`pwd`
GOIARDI_VERSION=`git describe --long --always`
GIT_HASH=`git rev-parse --short HEAD`
cd ..
gox -osarch="linux/amd64" -ldflags "-X github.com/ctdk/goiardi/config.GitHash=$GIT_HASH" -output="{{.Dir}}-$GOIARDI_VERSION-{{.OS}}-{{.Arch}}"

cd packaging/ubuntu/trusty
mkdir -p fs/usr/bin
mkdir -p fs/usr/share/goiardi
cp $CURDIR/../sql-files/*.sql fs/usr/share/goiardi
cp $CURDIR/README_GOIARDI_SCHEMA.txt fs/usr/share/goiardi
mkdir -p fs/var/lib/goiardi/lfs
cp ../../../goiardi-$GOIARDI_VERSION-linux-amd64 fs/usr/bin/goiardi

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C ./fs/ -p ../../artifacts/trusty/goiardi-VERSION_ARCH.deb -a amd64 --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cd ../../debian/wheezy
mkdir -p fs/usr/bin
mkdir -p fs/usr/share/goiardi
cp $CURDIR/../sql-files/*.sql fs/usr/share/goiardi
cp $CURDIR/README_GOIARDI_SCHEMA.txt fs/usr/share/goiardi
mkdir -p fs/var/lib/goiardi/lfs
cp ../../../goiardi-$GOIARDI_VERSION-linux-amd64 fs/usr/bin/goiardi
fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C ./fs/ -p ../../artifacts/wheezy/goiardi-VERSION_ARCH.deb -a amd64 --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cd ../../debian/jessie
mkdir -p fs/usr/bin
mkdir -p fs/usr/share/goiardi
cp $CURDIR/../sql-files/*.sql fs/usr/share/goiardi
cp $CURDIR/README_GOIARDI_SCHEMA.txt fs/usr/share/goiardi
mkdir -p fs/var/lib/goiardi/lfs
cp ../../../goiardi-$GOIARDI_VERSION-linux-amd64 fs/usr/bin/goiardi
fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C ./fs/ -p ../../artifacts/jessie/goiardi-VERSION_ARCH.deb -a amd64 --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cd ../../..

GOARM=6 gox -osarch="linux/arm" -ldflags "-X github.com/ctdk/goiardi/config.GitHash=$GIT_HASH" -output="{{.Dir}}-$GOIARDI_VERSION-{{.OS}}-{{.Arch}}6"
GOARM=7 gox -osarch="linux/arm" -ldflags "-X github.com/ctdk/goiardi/config.GitHash=$GIT_HASH" -output="{{.Dir}}-$GOIARDI_VERSION-{{.OS}}-{{.Arch}}7"

cd packaging/debian/wheezy
cp ../../../goiardi-$GOIARDI_VERSION-linux-arm6 fs/usr/bin/goiardi
fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C ./fs/ -p ../../artifacts/wheezy/goiardi-VERSION_ARCH.deb -a armel --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cp ../../../goiardi-$GOIARDI_VERSION-linux-arm7 fs/usr/bin/goiardi
fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C ./fs/ -p ../../artifacts/wheezy/goiardi-VERSION_ARCH.deb -a armhf --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

cd ../..
