#!/usr/bin/env bash

IP=35.193.197.105

#this is very big, don't deploy unnecessarily:
#scp -r acrcloud_linux/* ${IP}:acrcloud

scp -r ./frontend/* ${IP}:frontend
scp webserver.go ${IP}:
scp run.sh ${IP}:
scp soundcloud_dl.py ${IP}:
scp creds.sh ${IP}:
scp acr_recognise.py ${IP}:

ssh ${IP} <<'ENDSSH'
source creds.sh
export GOPATH=$(pwd)
go build webserver.go
sudo service webserver restart
ENDSSH