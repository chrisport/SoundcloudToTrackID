#!/usr/bin/env bash
#
# Example
# ./run https://soundcloud.com/elbuhoofficial/tecolotin
#python3 soundcloud_dl.py

source creds.sh
python3 acr_recognise.py "$1" $2
#rm "$filename"