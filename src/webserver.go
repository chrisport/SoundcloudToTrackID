package main

import (
	"net/http"
	"log"
	"os/exec"
	"fmt"
	"strings"
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
		log.Println(string(out))
		json := strings.Split(string(out),"\n")
		res := "unknown error occurred"
		for i:=len(json)-1;i>=0;i--{
			if json[i]!=""{
				res = json[i]
				break;
			}
		}
		rw.Write([]byte(res))
	})
	log.Fatal(http.ListenAndServe(":3000", nil))
}
