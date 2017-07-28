# SoundcloudToTrackID
Find TrackID of a track on soundcloud

The goal of this project is particularly to extrakt track ids out of DJ sets on soundcloud.   
Link + timestamp --> track id

For example:   
https://soundcloud.com/spak/melodias-de-los-bosques#t=25:51 --> Lemurian - 222

https://soundcloud.com/cosmotechmusic/cosmopods12-by-jati-div#t=38:00          -->?¿?
https://soundcloud.com/festivalnomade/peter-power-post-colonial-cafe#t=1:10:10 -->?¿?
https://soundcloud.com/pacayapacaya/lazy-acid#t=58:30                          -->?¿?

## Usage
workingdir is src folder   

### Webserver
running the webserver
```
    go run webserver.go
```
Requesting trackID
```
GET http://localhost:3000/recognise?url=<soundcloud url>&t=<timestamp>
```
supported timestamp formats are:
- 1h20m15s (= 80m20s = 4815s)
- 1:20:15

Example
```
GET http://localhost:3000/recognise?url=https://soundcloud.com/elbuhoofficial/tecolotin&t=1m20s
```

### Script
time in seconds
```
./run.sh https://soundcloud.com/elbuhoofficial/tecolotin  80
```

## TODO
- implement web interface
- avoid downloading whole file from soundcloud
- implement caching with timestamp-range
- automate deployment (terraform)
- port webserver into python
