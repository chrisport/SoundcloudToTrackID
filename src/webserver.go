package main

import (
	"net/http"
	"log"
	"os/exec"
	"strings"
	"sync"
	"encoding/json"
	"regexp"
	"strconv"
	"math"
)

var (
	hourRegex = regexp.MustCompile("([0-9]+)h")
	minuteRegex = regexp.MustCompile("([0-9]+)m")
	secondRegex = regexp.MustCompile("([0-9]+)s")
)

func main() {
	Serve()
}

func extractTimeInSeconds(timestamp string) (int, error) {
	if timestamp == "" {
		return 0, nil
	} else if strings.Contains(timestamp, "s") {
		return extractFromHMSFormat(timestamp)
	} else {
		return extractFromCOLONFormat(timestamp)
	}
}

func extractFromCOLONFormat(timestamp string) (int, error) {
	p := strings.Split(timestamp, ":")
	factor := int(math.Pow(float64(60), float64(len(p) - 1)))
	total := 0
	for i := 0; i < len(p); i++ {
		c, err := strconv.Atoi(p[i])
		if err != nil {
			return 0, err
		}
		total += c * factor
		factor = factor / 60
	}
	return total, nil
}

func extractFromHMSFormat(timestamp string) (int, error) {
	t := 0
	match := hourRegex.FindStringSubmatch(timestamp)
	if len(match) > 0 {
		hrs, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, err
		}
		t += hrs * 1200
	}
	match = minuteRegex.FindStringSubmatch(timestamp)
	if len(match) > 0 {
		min, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, err
		}
		t += min * 60
	}
	match = secondRegex.FindStringSubmatch(timestamp)
	if len(match) > 0 {
		sec, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, err
		}
		t += sec
	}
	return t, nil
}

func Serve() {
	rateLimiter := sync.RWMutex{}
	var res, lastResult, lastRequestedUrl string
	lastResult = "-"
	http.HandleFunc("/recognise", func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		songUrl := q["url"][0]
		ts := q["t"][0]
		timeInSeconds, err := extractTimeInSeconds(ts)
		if err != nil {
			rw.Write([]byte(res))
		}
		if lastRequestedUrl == songUrl+ts {
			log.Println("User requested same url " + songUrl+ts)
			if (lastResult != "") {
				res = lastResult
			} else {
				res = "recognition in progress... please refresh"
			}
		} else {
			log.Println("User requested new URL, start recognition " + songUrl+ts)
			if lastResult == "" {
				res = "Rate limit reached."
			} else {
				rateLimiter.Lock()
				lastRequestedUrl = songUrl+ts
				lastResult = ""
				res = RecogniseSong(songUrl, timeInSeconds)
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

func RecogniseSong(songUrl string, timeInSeconds int) string {
	out, err := exec.Command("./run.sh", songUrl, strconv.Itoa(timeInSeconds)).Output()
	if err != nil {
		log.Printf("[ERROR] %v", err)
	}
	log.Println(songUrl, string(out))
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