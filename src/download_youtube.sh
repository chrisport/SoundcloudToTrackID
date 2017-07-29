#!/usr/bin/env bash

source creds.sh

fileName=$(youtube-dl --extract-audio --audio-format mp3 --get-filename $1 | sed s/.webm/.mp3/g)
if [ ! -e "$fileName" ]; then
  youtube-dl --extract-audio --audio-format mp3 $1
fi
echo ${fileName}
