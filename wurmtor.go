package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/segment", SegmentHandler)

	port := ":8181"
	log.Printf("Started wurmtor on localhost%s\n", port)
	err := http.ListenAndServe(port, nil)

	if err != nil {
		log.Panic(err)
	}
}

func SegmentHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("Received segment from ", r.RemoteAddr)

	// Create file to store segment
	filename := "segment.go"
	file, err := os.Create(filename)
	if err != nil {
		log.Panic("Could not create file to store segment", err)
	}

	// Read from http POST
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic("Could not read body from http POST", err)
	}

	// Write segment to file
	file.Write(body)

	// Start command
	cmd := exec.Command("go", "run", filename)
	err = cmd.Start()
	if err != nil {
		log.Panic("Error starting segment", err)
	}

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	body := "Wurmtor running on " + hostname
	fmt.Fprintf(w, "<h1>%s</h1></br><p>Post segments to to /segment</p>", body)
}
