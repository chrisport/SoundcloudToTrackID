#!/usr/bin/env bash
#
# Example
# ./run https://soundcloud.com/elbuhoofficial/tecolotin
#
source creds.sh
op=$(python3 soundcloud_dl.py $1)
filename=${op##*$'\n'}
echo $filename