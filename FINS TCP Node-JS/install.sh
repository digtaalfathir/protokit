#!/bin/bash

set -e
set -x

if [ "$EUID" -ne 0 ]; then
	echo "Please run as root"
	exit 1
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

cp ${DIR}/plcdcs.service /etc/systemd/system
systemctl daemon-reload
systemctl enable plcdcs.service
systemctl start plcdcs.service

