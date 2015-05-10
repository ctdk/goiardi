#!/bin/sh

check_upstart_service(){
    status $1 | grep -q "^$1 start" > /dev/null
    return $?
}

set -e

[ -f /etc/default/goiardi ] && . /etc/default/goiardi

if ! getent group "$group" > /dev/null 2>&1 ; then
	addgroup --system "$group" --quiet
fi
if ! id $user > /dev/null 2>&1 ; then
	adduser --system --home /var/lib/goiardi --no-create-home \
		--ingroup "$group" --disabled-password --shell /bin/false \
		"$user"
fi
chgrp goiardi /etc/goiardi
chmod 775 /etc/goiardi
mkdir -p /var/log/goiardi
chown goiardi:goiardi /var/log/goiardi

if check_upstart_service goiardi; then
	restart goiardi
else
	start goiardi
fi


