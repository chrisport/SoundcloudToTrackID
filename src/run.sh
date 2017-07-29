#!/usr/bin/env bash
#
# Example
# ./run https://soundcloud.com/elbuhoofficial/tecolotin
#python3 soundcloud_dl.py
source creds.sh
op=$(python3 soundcloud_dl.py $1)
filename=${op##*$'\n'}
echo $filename
python3 test.py "$filename" $2
rm "$filename"