package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// mapping tile-x-y to ip.
var tile = map[string]string{
	"tile-0-0": "10.1.1.14",
	"tile-0-1": "10.1.1.15",
	"tile-0-2": "10.1.1.16",
	"tile-0-3": "10.1.1.20",
	"tile-1-0": "10.1.1.21",
	"tile-1-1": "10.1.1.22",
	"tile-1-2": "10.1.1.23",
	"tile-1-3": "10.1.1.24",
	"tile-2-0": "10.1.1.25",
	"tile-2-1": "10.1.1.26",
	"tile-2-2": "10.1.1.27",
	"tile-2-3": "10.1.1.28",
	"tile-3-0": "10.1.1.29",
	"tile-3-1": "10.1.1.30",
	"tile-3-2": "10.1.1.31",
	"tile-3-3": "10.1.1.32",
	"tile-4-0": "10.1.1.33",
	"tile-4-1": "10.1.1.34",
	"tile-4-2": "10.1.1.35",
	"tile-4-3": "10.1.1.36",
	"tile-5-0": "10.1.1.37",
	"tile-5-1": "10.1.1.38",
	"tile-5-2": "10.1.1.39",
	"tile-5-3": "10.1.1.40",
	"tile-6-0": "10.1.1.41",
	"tile-6-1": "10.1.1.42",
	"tile-6-2": "10.1.1.43",
	"tile-6-3": "10.1.1.44"}

var path string

func main() {

	path = *flag.String("path", "/tmp/wormgate",
		"where to store segment code")

	log.Printf("Changing working directory to " + path)
	os.Chdir(path)

	rand.Seed(time.Now().Unix())

	flag.Parse()

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/wormgate", WormGateHandler)
	http.HandleFunc("/segment", SegmentHandler)

	log.Println(tile)
	port := ":8181"
	log.Printf("Started wormgate on localhost%s\n", port)

	err := http.ListenAndServe(port, nil)

	if err != nil {
		log.Panic(err)
	}
}

func SegmentHandler(w http.ResponseWriter, r *http.Request) {

	// Read file received from segment
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}

	// Send the segment to every
	for _, v := range tile {

		load := make([]byte, len(body))
		n := copy(load, body)
		log.Println("Copied", n)

		reader := bytes.NewReader(load)

		url := "http://" + v + ":8181" + "/wormgate"

		log.Println("Sending segment to wormgate at", v)

		resp, err := http.Post(url, "string", reader)
		if err != nil {
			log.Println("POST error ", url, err)
		}

		log.Println(resp)
	}

	w.WriteHeader(http.StatusOK)

}

func WormGateHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("Received segment from wormgate at", r.RemoteAddr)

	// we'll extrackt and execute our segment in a new folder
	randomstring := fmt.Sprintf("%x", rand.Int63())
	extractionpath := path + "/" + randomstring
	filename := "tmp.tar.gz"
	fn := extractionpath + "/" + filename

	// Create directory to store segment code.
	err := os.MkdirAll(extractionpath, 0755)
	if err != nil {
		log.Panic("Could not create directory to store segment ", err)
	}
	os.Chdir(extractionpath)
	defer os.Chdir(path) // change back to base directory later

	// Create file and store incoming segment
	file, err := os.Create(fn)
	if err != nil {
		log.Panic("Could not create file to store segment", err)
	}
	defer os.Remove(fn) // let's remove the tarball later

	// Read from http POST
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic("Could not read body from http POST", err)
	}

	// Write tarball to file
	file.Write(body)
	err = file.Close()
	if err != nil {
		log.Panic("Error closing segment executable", err)
	}

	// extract segment
	tarCmd := exec.Command("tar", "-xzf", fn)

	log.Println(tarCmd)

	err = tarCmd.Run()
	if err != nil {
		log.Panic("Error extracting segment ", err)
	}

	// Start command, do not wait for it to complete
	binary := extractionpath + "/" + "hello-world-graphic"
	cmd := exec.Command(binary)
	//cmd.Dir = path
	err = cmd.Start()
	if err != nil {
		log.Panic("Error starting segment ", err)
	}

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	body := "Wormgate running on " + hostname
	fmt.Fprintf(w, "<h1>%s</h1></br><p>Post segments to to /segment</p>", body)
}
