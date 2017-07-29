package main

import (
	"net/http"
	"log"
	"os/exec"
	"strings"
	"regexp"
	"strconv"
	"math"
	"html/template"
	simplejson "github.com/bitly/go-simplejson"
	"github.com/chrisport/slotprovider"
	"sync"
	"fmt"
)

var (
	//TODO use file templates
	bootstrap = `<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
	<style> body {margin: 1em;}</style>`
	inProgressMessage = "<html><head>" + bootstrap + "<meta http-equiv=\"refresh\" content=\"2;\"></head><body>recognition in progress... you will be redirected automatically</body></html>"
	resultPage = `<html><head>` + bootstrap + `</head><body>
			{{.ErrorMessage}}<h1>{{if .Artist}}{{.Artist}} - {{.TrackName}}{{end}}<h1>
		        <button type="submit" class="btn btn-primary" onClick="window.history.go(-1); return false;">Search more</button>
			</body></html>`

	hourRegex = regexp.MustCompile("([0-9]+)h")
	minuteRegex = regexp.MustCompile("([0-9]+)m")
	secondRegex = regexp.MustCompile("([0-9]+)s")
	cleanUrlRegex = regexp.MustCompile("(.*)\\?")
)

func main() {
	Serve()
}

func extractTimeInSeconds(timestamp string) (int, error) {
	if timestamp == "" {
		return 0, nil
	} else if strings.Contains(timestamp, "s") || strings.Contains(timestamp, "h") || strings.Contains(timestamp, "m") {
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
		songUrl = cleanUrl(songUrl)
		fmt.Println(songUrl)
		fmt.Println(songUrl)
		fmt.Println(songUrl)
		fmt.Println(songUrl)
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

func cleanUrl(url string) string {
	if strings.Contains(url, "?") {
		url = cleanUrlRegex.FindStringSubmatch(url)[0]
		url = url[:len(url)-1]
	}
	return url
}

type Result struct {
	ErrorMessage string
	Artist       string
	TrackName    string
	RawBody      string
}

func newThrottledRecogniser() func(songUrl string, ts string) *Result {
	recognitionSP := slotprovider.New(5)
	return func(songUrl, ts string) *Result {
		timeInSeconds, err := extractTimeInSeconds(ts)
		if err != nil {
			return &Result{ErrorMessage:err.Error()}
		}

		fullUrl := songUrl + strconv.Itoa(timeInSeconds)
		initialized, res := getFromCache(fullUrl)
		if res != nil {
			log.Printf("Responding to %v with cached result", fullUrl)
			return res
		} else if initialized {
			// this item is processing currently
			log.Printf("Responding to %v with 'in progress'", fullUrl)
			return &Result{RawBody:inProgressMessage}
		}
		//else start recognition

		acquired, release := recognitionSP.AcquireSlot()
		if !acquired {
			log.Printf("Responding to %v with 'no free slots'", fullUrl)
			return &Result{ErrorMessage:"Request limit reached. We are not able to recognize more songs at the moment. Please try later."}
		}

		go func() {
			log.Printf("Start recognition of %v", fullUrl)
			reserveCache(fullUrl)
			result := RecogniseSong(songUrl, timeInSeconds)
			res := parseResult(result)
			release()
			putResultToCache(fullUrl, res)
		}()
		log.Printf("Responding to %v with 'in progress' and start recognition", fullUrl)
		return &Result{RawBody:inProgressMessage}
	}

}

func parseResult(result string) *Result {
	sj, err := simplejson.NewJson([]byte(result))
	if err != nil {
		return &Result{ErrorMessage:"Error occurred"}
	}

	noResult, err := (*sj).GetPath("status", "msg").String()
	if noResult == "No result" {
		return &Result{ErrorMessage:"Track could not be recognised."}
	}

	artist, err := (*sj).GetPath("metadata", "music").GetIndex(0).Get("artists").GetIndex(0).Get("name").String()
	if err != nil {
		return &Result{ErrorMessage:"Error occurred"}
	}

	trackName, err := (*sj).GetPath("metadata", "music").GetIndex(0).Get("title").String()
	if err != nil {
		return &Result{ErrorMessage:"Error occurred"}
	} else {

		return &Result{Artist:artist, TrackName:trackName}
	}
}

func RecogniseSong(songUrl string, timeInSeconds int) string {
	log.Println("./run.sh", songUrl, strconv.Itoa(timeInSeconds))
	out, err := exec.Command("./run.sh", songUrl, strconv.Itoa(timeInSeconds)).Output()
	if err != nil {
		log.Printf("[ERROR] %v", err)
	}
	log.Println(songUrl, "Result received for " + songUrl)
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

var (
	cache = make(map[string]*Result)
	cacheMux = sync.Mutex{}
	InitialResult = &Result{ErrorMessage:"Processing"}
)

func reserveCache(id string) {
	cacheMux.Lock()
	defer cacheMux.Unlock()
	cache[id] = InitialResult
}

func putResultToCache(id string, result *Result) {
	cacheMux.Lock()
	defer cacheMux.Unlock()
	if cache[id] == nil {
		log.Printf("Warning: Result for %v has been stored in cache without prior reservation\n", id)
	}
	cache[id] = result
}

func getFromCache(id string) (initialized bool, result *Result) {
	cacheMux.Lock()
	defer cacheMux.Unlock()
	existing := cache[id]
	if existing == nil {
		return false, nil
	}
	if existing == InitialResult {
		return true, nil
	}
	return true, existing
}