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
	"encoding/json"
	"os"
	"io/ioutil"
	"time"
)
//TODO proper project setup, modular separation
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
	notRecognisedMessage = "Track could not be recognised."
)

func main() {
	loadCacheFromDisc()
	go func() {
		for {
			time.Sleep(20 * time.Second)
			saveCacheToDisc()
		}
	}()
	Serve()
}

type Result struct {
	ErrorMessage string `json:"error,omitempty"`
	Artist       string `json:"artist,omitempty"`
	TrackName    string `json:"trackName,omitempty"`
	RawBody      string `json:"rawBody,omitempty"`
}

func Serve() {
	fs := http.FileServer(http.Dir("frontend"))
	http.Handle("/", fs)
	http.HandleFunc("/stats", func(rw http.ResponseWriter, req *http.Request) {
		dump, err := dumpCache()
		if err != nil {
			rw.Write([]byte(err.Error()))
		}
		rw.Write(dump)
	})
	throttledRecogniser := newThrottledRecogniser()

	http.HandleFunc("/api/recognise", func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		songUrl := q["url"][0]
		songUrl = cleanUrl(songUrl)
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

func newThrottledRecogniser() func(songUrl string, ts string) *Result {
	recognitionSP := slotprovider.New(5)
	return func(songUrl, ts string) *Result {
		timeInSeconds, err := extractTimeInSeconds(ts)
		if err != nil {
			return &Result{ErrorMessage:err.Error()}
		}
		timeInSeconds = floorToInterval(timeInSeconds, 30)
		fullUrl := songUrl + "#t=" + strconv.Itoa(timeInSeconds) + "s"
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

		isSlotAcquired, releaseSlot := recognitionSP.AcquireSlot()
		if !isSlotAcquired {
			log.Printf("Responding to %v with 'no free slots'", fullUrl)
			return &Result{ErrorMessage:"Request limit reached. We are not able to recognize more songs at the moment. Please try later."}
		}
		reserveCache(fullUrl)

		go func() {
			defer releaseSlot()
			log.Printf("Start recognition of %v", fullUrl)
			result := RecogniseSong(songUrl, timeInSeconds)
			res := parseResult(result)
			putResultToCache(fullUrl, res)
		}()
		log.Printf("Responding to %v with 'in progress' and start recognition", fullUrl)
		return &Result{RawBody:inProgressMessage}
	}

}

func floorToInterval(time int, intervall int) int {
	fx := float32(time) / float32(intervall)
	f := time / intervall
	if fx - float32(f) < 0.5 {
		return f * intervall
	} else {
		return (f + 1) * intervall
	}
}

func parseResult(result string) *Result {
	sj, err := simplejson.NewJson([]byte(result))
	if err != nil {
		return &Result{ErrorMessage:"Error occurred"}
	}

	noResult, err := (*sj).GetPath("status", "msg").String()
	if noResult == "No result" {
		return &Result{ErrorMessage:notRecognisedMessage}
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

func RecogniseSong(songUrl string, timeInSeconds int) (string) {
	filePath, err := downloadSong(songUrl)
	if err != nil {
		return "Error while downloading: " + err.Error()
	}
	result, err := sendSongToACR(filePath, timeInSeconds)
	if err != nil {
		return "Error while recognizing: " + err.Error()
	}
	return result
}

func downloadSong(songUrl string) (string, error) {
	var fileName string
	var err error
	if strings.Contains(songUrl, "youtube") {
		fileName, err = executeAndGetLastLine("./download_youtube.sh", songUrl)
		if err != nil {
			return "", err
		}
	} else if strings.Contains(songUrl, "soundcloud") {
		fileName, err = executeAndGetLastLine("./download_soundcloud.sh", songUrl)
		if err != nil {
			return "", err
		}
	}

	log.Println("Downloaded song to file: ", fileName, songUrl)
	return fileName, nil
}

func sendSongToACR(filePath string, timeInSeconds int) (string, error) {
	return executeAndGetLastLine("./recognise.sh", filePath, strconv.Itoa(timeInSeconds))
}

func executeAndGetLastLine(script string, opts... string) (string, error) {
	out, err := exec.Command(script, opts...).Output()
	if err != nil {
		log.Printf("[ERROR] Script %v failed with %v\n", script, err)
		return "", err
	}
	log.Printf("[SUCCESS] Script %v finished\n", script)
	return getLastLine(string(out)), nil
}

// ### HELPER ####
func getLastLine(input string) string {
	lines := strings.Split(string(input), "\n")
	res := "unknown error occurred"
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			res = lines[i]
			break;
		}
	}
	return res
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

func cleanUrl(url string) string {
	if strings.Contains(url, "soundcloud") && strings.Contains(url, "?") {
		url = cleanUrlRegex.FindStringSubmatch(url)[0]
		url = url[:len(url) - 1]
	}
	return url
}



// ############ Cache ############
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

func dumpCache() ([]byte, error) {
	cacheMux.Lock()
	defer cacheMux.Unlock()
	return json.Marshal(cache)
}

const cacheDumpFilePath = "cachedump.json"

func loadCacheFromDisc() {
	if content, err := ioutil.ReadFile(cacheDumpFilePath); err == nil {
		var savedCache map[string]*Result
		err := json.Unmarshal(content, &savedCache)
		if err != nil {
			log.Println(err)
			return
		}
		keysToDelete := make([]string, 0)
		for k, v := range savedCache {
			if v.ErrorMessage != "" && v.ErrorMessage != notRecognisedMessage {
				keysToDelete = append(keysToDelete, k)
			}
		}
		for _, k := range keysToDelete {
			delete(savedCache, k)
		}
		cacheMux.Lock()
		defer cacheMux.Unlock()
		cache = savedCache
	} else if !os.IsNotExist(err) {
		log.Println("Could not load dump", err)
	}
}

func saveCacheToDisc() {
	log.Println("Saving result dump to disk")
	var f *os.File
	var err error

	f, err = os.OpenFile(cacheDumpFilePath, os.O_CREATE | os.O_RDWR | os.O_TRUNC, 0666)
	defer f.Close()

	bytes, err := dumpCache()
	if err != nil {
		log.Println(err)
		return
	}
	f.Write(bytes)
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