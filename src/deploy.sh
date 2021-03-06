#!/usr/bin/env bash

source creds.sh

#this is very big, don't deploy unnecessarily:
#scp -r acrcloud_linux/* ${SN_IP}:acrcloud

scp -r ./frontend/* ${SN_IP}:frontend
scp webserver.go ${SN_IP}:
scp *.sh ${SN_IP}:

ssh ${SN_IP} <<'ENDSSH'
source creds.sh
export GOPATH=$(pwd)
go build webserver.go
sudo service webserver restart
ENDSSH