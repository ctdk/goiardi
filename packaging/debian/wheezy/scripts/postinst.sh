#!/bin/sh

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
chown -R $user:$group /var/lib/goiardi
mkdir -p /var/log/goiardi
chown $user:$group /var/log/goiardi

if [ -x /bin/systemctl ]; then
	/bin/systemctl daemon-reload
	/bin/systemctl start goiardi.service
else 
	update-rc.d -f goiardi defaults
	invoke-rc.d goiardi start
fi
