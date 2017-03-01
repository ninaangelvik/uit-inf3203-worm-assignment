package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {

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
		// TODO
	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}

func sendSegment(address string) {

	port := ":8181"
	url := "http://" + address + port + "/segment"
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

	log.Println("repsonse:", resp)
}
