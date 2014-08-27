package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var path string

func main() {

	path = *flag.String("path", "/tmp/wormgate",
		"where to store incoming segments")

	flag.Parse()

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

	filename := "segment"
	fn := path + "/" + filename

	// Create directory to store segment code.
	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Panic("Could not create directory to store segment ", err)
	}

	// Create file and store incoming segment
	file, err := os.Create(fn)
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

	err = file.Close()
	if err != nil {
		log.Panic("Error closing segment executable", err)
	}

	// Make segment code executable
	cmd := exec.Command("chmod", "+x", fn)
	err = cmd.Run()
	if err != nil {
		log.Panic("Error making segment code executable ", err)
	}

	// Start command, do not wait for it to complete
	cmd = exec.Command(fn)
	//cmd.Dir = path
	err = cmd.Start()
	if err != nil {
		log.Panic("Error starting segment ", err)
	}

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	body := "Wurmtor running on " + hostname
	fmt.Fprintf(w, "<h1>%s</h1></br><p>Post segments to to /segment</p>", body)
}
