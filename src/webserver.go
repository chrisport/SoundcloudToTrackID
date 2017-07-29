package main

import (
	"net/http"
	"log"
	"os/exec"
	"strings"
	"sync"
	"regexp"
	"strconv"
	"math"
	"html/template"
	simplejson "github.com/bitly/go-simplejson"
)

var (
	//TODO use file tempaltes
	bootstrap = `<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
	<style> body {margin: 1em;}</style>`
	inProgressMessage = "<html><head>" + bootstrap + "<meta http-equiv=\"refresh\" content=\"2;\"></head><body>recognition in progress... you will be redirected automatically</body></html>"
	resultPage = `<html><head>` + bootstrap + `</head><body>
			<h1>{{.ErrorMessage}}{{if .Artist}}{{.Artist}} - {{.TrackName}}{{end}}<h1>
		        <button type="submit" class="btn btn-primary" onClick="window.history.go(-1); return false;">Search more</button>
			</body></html>`

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
	fs := http.FileServer(http.Dir("frontend"))
	http.Handle("/", fs)
	throttledRecogniser := newThrottledRecogniser()
	http.HandleFunc("/api/recognise", func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		songUrl := q["url"][0]
		ts := q["t"][0]
		result := throttledRecogniser(songUrl, ts)

		if result.RawBody != "" {
			rw.Write([]byte(result.RawBody))
			return
		}

		t := template.New("some template") // Create a template.
		//t2, err := t.ParseFiles("./frontend/result_page.html")  // Parse template file.
		t2, err := t.Parse(resultPage)  // Parse template file.
		if err != nil {
			panic(err)
		}

		t2.Execute(rw, *result)
	})
	log.Fatal(http.ListenAndServe(":3000", nil))
}

type Response struct {
	ErrorMessage string
	Artist       string
	TrackName    string
	RawBody      string
}

func newThrottledRecogniser() func(songUrl string, ts string) *Response {
	rateLimiter := sync.RWMutex{}
	var lastRequestedUrl string
	var lastResult = &Response{}
	return func(songUrl, ts string) *Response {
		timeInSeconds, err := extractTimeInSeconds(ts)
		if err != nil {
			return &Response{ErrorMessage:err.Error()}

		}
		if lastRequestedUrl == songUrl + ts {
			log.Println("User requested same url " + songUrl + ts)
			if (lastResult != nil) {
				return lastResult
			} else {
				return &Response{RawBody:inProgressMessage}
			}
		} else {
			log.Println("User requested new URL, start recognition " + songUrl + ts)
			if lastResult == nil {
				return &Response{ErrorMessage:"Rate limit reached."}
			} else {
				rateLimiter.Lock()
				go func() {
					lastRequestedUrl = songUrl + ts
					lastResult = nil
					result := RecogniseSong(songUrl, timeInSeconds)
					res := parseResult(result)
					lastResult = res
					rateLimiter.Unlock()
				}()
				return &Response{RawBody:inProgressMessage}
			}

		}
	}

}

func parseResult(result string) *Response {
	sj, err := simplejson.NewJson([]byte(result))
	if err != nil {
		return &Response{ErrorMessage:"Error occurred"}
	}

	noResult, err := (*sj).GetPath("status", "msg").String()
	if noResult == "No result" {
		return &Response{ErrorMessage:"Track could not be recognised."}
	}

	artist, err := (*sj).GetPath("metadata", "music").GetIndex(0).Get("artists").GetIndex(0).Get("name").String()
	if err != nil {
		return &Response{ErrorMessage:"Error occurred"}
	}

	trackName, err := (*sj).GetPath("metadata", "music").GetIndex(0).Get("title").String()
	if err != nil {
		return &Response{ErrorMessage:"Error occurred"}
	} else {

		return &Response{Artist:artist, TrackName:trackName}
	}
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