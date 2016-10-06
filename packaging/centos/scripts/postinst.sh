#!/bin/sh

set -e

[ -f /etc/sysconfig/goiardi ] && . /etc/sysconfig/goiardi

if ! getent group "$group" > /dev/null 2>&1 ; then
	groupadd -r "$group"
fi
if ! getent passwd "$user" > /dev/null 2>&1 ; then
	useradd -r -g $group -d /var/lib/goiardi -s /sbin/nologin -c "goiardi user" $user
fi

chgrp $group /etc/goiardi
chmod 775 /etc/goiardi
chown -R $user:$group /var/lib/goiardi
mkdir -p /var/log/goiardi
chown $user:$group /var/log/goiardi
