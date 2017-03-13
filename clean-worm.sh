#!/bin/bash
USER=$(whoami)
set -x
./ssh-all.sh killall -u $USER
./ssh-all.sh rm -r /tmp/wormgate-$USER
