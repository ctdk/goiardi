#!/bin/sh

# For now, this will build debs for ubuntu 14.04 and debian wheezy on amd64.
# Requires gox and fpm to be installed.

# make more easily specified later
GOIARDI_VERSION="0.10.0"
ITERATION="1ubuntu1"

cd ..
gox -osarch="linux/amd64" -output="{{.Dir}}-$GOIARDI_VERSION-{{.OS}}-{{.Arch}}"

cd packaging/ubuntu/trusty
mkdir -p fs/usr/bin
cp ../../../goiardi-$GOIARDI_VERSION-linux-amd64 fs/usr/bin/goiardi

fpm -s dir -t deb -n goiardi -v $GOIARDI_VERSION --iteration $ITERATION -C ./fs/ -p goiardi-VERSION-ITERATION_ARCH.deb -a amd64 --description "a golang chef server" --after-install ./scripts/postinst.sh --after-remove ./scripts/postrm.sh --deb-suggests mysql-server --deb-suggests postgresql --license apachev2 -m "<jeremy@goiardi.gl>" .
