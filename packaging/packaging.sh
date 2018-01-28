#!/bin/bash

# For now, this will build debs for ubuntu 14.04 and debian wheezy on amd64.
# Requires gox and fpm to be installed.

# make more easily specified later
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

# resync tags, if needed
git pull --tags

CURDIR=`pwd`
ARTIFACT_DIR=$CURDIR/artifacts
GOIARDI_VERSION=`git describe --long --always`
GIT_HASH=`git rev-parse --short HEAD`
COMMON_DIR="$CURDIR/common"
BUILD="$CURDIR/build"
SHARE="$BUILD/share"
DOCS=$SHARE/docs
MAN=$BUILD/man

rm -r $BUILD
rm -r $ARTIFACT_DIR

for VAR in trusty wheezy jessie el6 el7; do
	mkdir -p $ARTIFACT_DIR/$VAR
done

mkdir -p $BUILD/bin
mkdir $SHARE
cp $CURDIR/../sql-files/*.sql $SHARE
cp $CURDIR/README_GOIARDI_SCHEMA.txt $SHARE

cd ..
go build
gox -osarch="linux/amd64 linux/armv6 linux/armv7 linux/s390x" -ldflags "-X github.com/ctdk/goiardi/config.GitHash=$GIT_HASH" -output="$BUILD/{{.Dir}}-$GOIARDI_VERSION-{{.OS}}-{{.Arch}}"

mkdir $MAN
./goiardi --print-man-page | sed 's/\.TH goiardi 1/\.TH goiardi 8/' > $MAN/goiardi.8
cd docs
make html
cp -r _build/html $DOCS

BUILD_ROOT="$BUILD/trusty"
FILES_DIR="$CURDIR/ubuntu/trusty"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-amd64 usr/bin/goiardi
cp -r $FILES_DIR/fs/etc .
cp -r $COMMON_DIR/* .

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/trusty/goiardi-VERSION_ARCH.deb -a amd64 --description "a golang chef server" --after-install $FILES_DIR/scripts/postinst.sh --after-remove $FILES_DIR/scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

BUILD_ROOT="$BUILD/wheezy"
FILES_DIR="$CURDIR/debian/wheezy"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-amd64 usr/bin/goiardi
cp -r $FILES_DIR/fs/etc .
cp -r $COMMON_DIR/* .

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/wheezy/goiardi-VERSION_ARCH.deb -a amd64 --description "a golang chef server" --after-install $FILES_DIR/scripts/postinst.sh --after-remove $FILES_DIR/scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

BUILD_ROOT="$BUILD/jessie"
FILES_DIR="$CURDIR/debian/jessie"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-amd64 usr/bin/goiardi
cp -r $FILES_DIR/fs/lib .
cp -r $COMMON_DIR/* .

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/jessie/goiardi-VERSION_ARCH.deb -a amd64 --description "a golang chef server"  --after-install $FILES_DIR/scripts/postinst.sh --after-remove $FILES_DIR/scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

# CentOS

CENTOS_COMMON_DIR="$CURDIR/centos/common"
CENTOS_SCRIPTS="$CURDIR/centos/scripts"

BUILD_ROOT="$BUILD/el6"
FILES_DIR="$CURDIR/centos/6"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-amd64 usr/bin/goiardi
cp -r $FILES_DIR/fs/etc .
cp -r $COMMON_DIR/* .
cp -r $CENTOS_COMMON_DIR/etc .

fpm -s dir -t rpm -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/el6/goiardi-VERSION.el6.ARCH.rpm -a amd64 --description "a golang chef server" --after-install $CENTOS_SCRIPTS/postinst.sh --license apachev2 -m "<jeremy@goiardi.gl>" .

BUILD_ROOT="$BUILD/el7"
FILES_DIR="$CURDIR/debian/jessie"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-amd64 usr/bin/goiardi
cp -r $FILES_DIR/fs/lib .
cp -r $COMMON_DIR/* .
cp -r $CENTOS_COMMON_DIR/etc .

fpm -s dir -t rpm -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/el7/goiardi-VERSION.el7.ARCH.rpm -a amd64 --description "a golang chef server" --after-install $CENTOS_SCRIPTS/postinst.sh --license apachev2 -m "<jeremy@goiardi.gl>" .

# ARM binaries

cd $CURDIR
cd ..

BUILD_ROOT="$BUILD/wheezy-armv6"
FILES_DIR="$CURDIR/debian/wheezy"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-armv6 usr/bin/goiardi
cp -r $FILES_DIR/fs/etc .
cp -r $COMMON_DIR/* .

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/wheezy/goiardi-VERSION_ARCH.deb -a armel --description "a golang chef server" --after-install $FILES_DIR/scripts/postinst.sh --after-remove $FILES_DIR/scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

BUILD_ROOT="$BUILD/wheezy-armv7"
FILES_DIR="$CURDIR/debian/wheezy"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-armv7 usr/bin/goiardi
cp -r $FILES_DIR/fs/etc .
cp -r $COMMON_DIR/* .

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/wheezy/goiardi-VERSION_ARCH.deb -a armhf --description "a golang chef server" --after-install $FILES_DIR/scripts/postinst.sh --after-remove $FILES_DIR/scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

BUILD_ROOT="$BUILD/jessie-armv6"
FILES_DIR="$CURDIR/debian/jessie"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-armv6 usr/bin/goiardi
cp -r $FILES_DIR/fs/lib .
cp -r $COMMON_DIR/* .

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/jessie/goiardi-VERSION_ARCH.deb -a armel --description "a golang chef server"  --after-install $FILES_DIR/scripts/postinst.sh --after-remove $FILES_DIR/scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

BUILD_ROOT="$BUILD/jessie-armv7"
FILES_DIR="$CURDIR/debian/jessie"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-armv7 usr/bin/goiardi
cp -r $FILES_DIR/fs/lib .
cp -r $COMMON_DIR/* .

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/jessie/goiardi-VERSION_ARCH.deb -a armhf --description "a golang chef server"  --after-install $FILES_DIR/scripts/postinst.sh --after-remove $FILES_DIR/scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

BUILD_ROOT="$BUILD/jessie-s390x"
FILES_DIR="$CURDIR/debian/jessie"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-s390x usr/bin/goiardi
cp -r $FILES_DIR/fs/lib .
cp -r $COMMON_DIR/* .

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/jessie/goiardi-VERSION_ARCH.deb -a s390x --description "a golang chef server"  --after-install $FILES_DIR/scripts/postinst.sh --after-remove $FILES_DIR/scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .

BUILD_ROOT="$BUILD/el7-s390x"
FILES_DIR="$CURDIR/debian/jessie"
mkdir -p $BUILD_ROOT
cd $BUILD_ROOT
mkdir -p usr/bin
mkdir -p usr/share/goiardi
mkdir -p usr/share/man/man8
cp $MAN/goiardi.8 usr/share/man/man8
cp -r $SHARE/* usr/share/goiardi
mkdir -p var/lib/goiardi/lfs
cp $BUILD/goiardi-$GOIARDI_VERSION-linux-s390x usr/bin/goiardi
cp -r $FILES_DIR/fs/lib .
cp -r $COMMON_DIR/* .
cp -r $CENTOS_COMMON_DIR/etc .

fpm -s dir -t rpm -n goiardi -v $GOIARDI_VERSION -C . -p $ARTIFACT_DIR/el7/goiardi-VERSION.el7.ARCH.rpm -a s390x --description "a golang chef server" --after-install $CENTOS_SCRIPTS/postinst.sh --license apachev2 -m "<jeremy@goiardi.gl>" .

cd $CURDIR
