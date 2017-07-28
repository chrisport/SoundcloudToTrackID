package main

import (
	"net/http"
	"log"
	"os/exec"
	"strings"
	"sync"
	"encoding/json"
)

func main() {
	Serve()
}

func Serve() {
	rateLimiter := sync.RWMutex{}
	var res, lastResult, lastRequestedUrl string
	lastResult = "-"
	http.HandleFunc("/recognise", func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		songUrl := q["url"][0]
		timestamp := q["t"][0]
		if timestamp == ""{
			timestamp = "0"
		}
		if lastRequestedUrl == songUrl {
			log.Println("User requested same url " + songUrl)
			if (lastResult != "") {
				res = lastResult
			} else {
				res = "recognition in progress... please refresh"
			}
		} else {
			log.Println("User requested new URL, start recognition " + songUrl)
			if lastResult == "" {
				res = "Rate limit reached."
			} else {
				rateLimiter.Lock()
				lastRequestedUrl = songUrl
				lastResult = ""
				res = RecogniseSong(songUrl,timestamp)
				var asd interface{}
				err := json.Unmarshal([]byte(res), &asd)
				if err != nil {
					res = "Error occurred"
				}
				lastResult = res
				rateLimiter.Unlock()
			}

		}

		rw.Write([]byte(res))
	})
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func RecogniseSong(songUrl string, timestamp string) string {
	out, err := exec.Command("./run.sh", songUrl,timestamp).Output()
	if err != nil {
		log.Printf("[ERROR] %v", err)
	}
	log.Println(songUrl,string(out))
	lines := strings.Split(string(out), "\n")
	res := "unknown error occurred"
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			res = lines[i]
			break;
		}
	}
	return res
}