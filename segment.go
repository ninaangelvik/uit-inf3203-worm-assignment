package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const wormgatePort = ":8181"
const segmentPort = ":8182"

var hostname string

func main() {

	hostname, _ = os.Hostname()
	log.SetPrefix(hostname + " segment: ")

	var spreadMode = flag.NewFlagSet("spread", flag.ExitOnError)
	var spreadHost = spreadMode.String("host", "localhost", "host to spread to")

	if len(os.Args) == 1 {
		log.Fatalf("No mode specified\n")
	}

	switch os.Args[1] {
	case "spread":
		spreadMode.Parse(os.Args[2:])
		sendSegment(*spreadHost)
	case "run":
		startSegmentServer()

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}

func sendSegment(address string) {

	url := "http://" + address + wormgatePort + "/wormgate"
	filename := "tmp.tar.gz"

	log.Printf("Spreading to %s", url)

	// ship the binary and the qml file that describes our screen output
	tarCmd := exec.Command("tar", "-zc", "-f", filename, "segment")
	tarCmd.Run()
	defer os.Remove(filename)

	file, err := os.Open(filename)
	if err != nil {
		log.Panic("Could not read input file", err)
	}

	resp, err := http.Post(url, "string", file)
	if err != nil {
		log.Panic("POST error ", err)
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Println("Received OK from server")
	} else {
		log.Println("Response: ", resp)
	}
}

func startSegmentServer() {
	http.HandleFunc("/", IndexHandler)

	log.Printf("Starting segment server on %s%s\n", hostname, segmentPort)
	err := http.ListenAndServe(segmentPort, nil)
	if err != nil {
		log.Panic(err)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	// We don't use the body, but read it anyway
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	body := "Segment running on " + hostname
	fmt.Fprintf(w, "<h1>%s</h1></br><p>No further instructions</p>", body)
}
