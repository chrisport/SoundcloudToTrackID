Unfortunately soundsniffer.chrisport.ch is currently offline, due to expiration of Google Cloud Platform trial. Redeployment might be considered in the future.

# SoundcloudToTrackID
Find TrackID of a track on soundcloud

The goal of this project is particularly to extrakt track ids out of DJ sets on soundcloud.   
Link + timestamp --> track id

For example:   
https://soundcloud.com/spak/melodias-de-los-bosques#t=25:51    
    --> Lemurian - 222   
or:   
https://www.youtube.com/watch?v=YDWEz1mia1I and t=9m   
    --> Matt Elliott - C.f. bundy   
 
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
- avoid downloading whole file from soundcloud
- automate deployment (terraform)
