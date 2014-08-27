package main

import (
	"log"
	"net/http"
	"os"
        "os/exec"
)

func main() {

	address := "http://localhost"
	port := ":8181"
	url := address + port + "/segment"
	filename := "tmp.tar.gz"

        // ship the binary and the qml file that describes our screen output
        tarCmd := exec.Command("tar", "-zc", "-f" + filename,
            "hello-world-graphic", "hello-world.qml")
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
