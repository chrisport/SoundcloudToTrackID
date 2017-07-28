#!/usr/bin/env bash
#
# Example
# ./run https://soundcloud.com/elbuhoofficial/tecolotin
#python3 soundcloud_dl.py
op=$(python3 soundcloud_dl.py $1 $2)
filename=${op##*$'\n'}
echo $filename
python3 test.py "$filename"
rm "$filename"