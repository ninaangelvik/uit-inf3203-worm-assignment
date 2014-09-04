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
        "time"
)

var path string

func main() {

	path = *flag.String("path", "/tmp/wormgate",
		"where to store incoming segments")

        log.Printf("Changing working directory to " + path)
        os.Chdir(path)

        rand.Seed(time.Now().Unix())
        //log.Printf(hex.EncodeToString(rand.Int63()))

	flag.Parse()

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/segment", SegmentHandler)

	port := ":8181"
	log.Printf("Started wormgate on localhost%s\n", port)
	err := http.ListenAndServe(port, nil)

	if err != nil {
		log.Panic(err)
	}
}

func SegmentHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("Received segment from ", r.RemoteAddr)

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
        defer os.Chdir(path)  // change back to base directory later

	// Create file and store incoming segment
	file, err := os.Create(fn)
	if err != nil {
		log.Panic("Could not create file to store segment", err)
	}
        defer os.Remove(fn)  // let's remove the tarball later

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
        tarCmd := exec.Command("tar", "-x", "-f" + fn)
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
