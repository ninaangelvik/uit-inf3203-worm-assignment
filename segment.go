package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
		startPayload()
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
	tarCmd := exec.Command("tar", "-zc", "-f"+filename,
		"segment", "payload")
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

	if resp.StatusCode == 200 {
		log.Println("Received OK from server")
	} else {
		log.Println("Response: ", resp)
	}
	resp.Body.Close()
}

func startPayload() {
	// Start payload, do not wait for it to complete
	var dir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Panic("Error starting payload: ", err)
	}

	binary := dir + "/" + "payload"
	cmdline := []string{"stdbuf", "-oL", "-eL", binary}
	log.Printf("Running payload: %q", cmdline)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()

	if err != nil {
		log.Panic("Error starting payload: ", err)
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
	body := "Segment running on " + hostname
	fmt.Fprintf(w, "<h1>%s</h1></br><p>No further instructions</p>", body)
}
