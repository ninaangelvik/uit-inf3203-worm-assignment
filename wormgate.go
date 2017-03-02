package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"time"
)

const wormgatePort = ":8181"
const segmentPort = ":8182"

var path string

func main() {

	var hostname, _ = os.Hostname()
	log.SetPrefix(hostname + " wormgate: ")

	curuser, err := user.Current()
	if err!=nil {
		log.Panic(err)
	}
	log.Printf("Current user: %s\n", curuser.Username)

	path = *flag.String("path", "/tmp/wormgate-" + curuser.Username,
		"where to store segment code")

	log.Printf("Changing working directory to " + path)
	os.Chdir(path)

	rand.Seed(time.Now().Unix())

	flag.Parse()

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/wormgate", WormGateHandler)

	log.Printf("Started wormgate on %s%s\n", hostname, wormgatePort)

	err = http.ListenAndServe(wormgatePort, nil)

	if err != nil {
		log.Panic(err)
	}
}

func WormGateHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("Received segment from", r.RemoteAddr)

	// we'll extract and execute our segment in a new folder
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
	r.Body.Close()

	// Write tarball to file
	file.Write(body)
	err = file.Close()
	if err != nil {
		log.Panic("Error closing segment executable", err)
	}

	// extract segment
	cmdline := []string{"tar", "-xzf", fn}
	log.Printf("Extracting segment: %q", cmdline)
	tarCmd := exec.Command(cmdline[0], cmdline[1:]...)
	err = tarCmd.Run()
	if err != nil {
		log.Panic("Error extracting segment ", err)
	}

	// Start command, do not wait for it to complete
	binary := extractionpath + "/" + "segment"
	cmdline = []string{"stdbuf", "-oL", "-eL", binary, "run"}
	log.Printf("Running segment: %q", cmdline)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
