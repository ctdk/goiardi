#!/bin/sh
/bin/systemctl stop goiardi
/bin/systemctl disable goiardi
/bin/systemctl daemon-reload
