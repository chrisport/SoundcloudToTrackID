package main

import (
	"net/http"
	"log"
	"os/exec"
	"fmt"
)

func main() {
	Serve()
}

func Serve() {
	http.HandleFunc("/recognise", func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		songUrl := q["url"][0]
		fmt.Println("received request for ",songUrl)
		out, err := exec.Command("./run.sh", songUrl).Output()
		if err != nil {
			fmt.Errorf("%v",err)
		}
		rw.Write(out)
	})
	log.Fatal(http.ListenAndServe(":3000", nil))
}
