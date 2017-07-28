#!/usr/bin/env bash

IP=35.193.197.105
#scp -r acrcloud_linux/* ${IP}:acrcloud
scp -r ./frontend/* ${IP}:frontend
scp webserver.go ${IP}:
scp run.sh ${IP}:
scp soundcloud_dl.py ${IP}:
scp test.py ${IP}:

ssh ${IP} <<'ENDSSH'
sudo service webserver restart
ENDSSH