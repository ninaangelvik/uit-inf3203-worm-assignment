package main

import (
	"log"
	"net/http"
	"os"
)

func main() {

	address := "http://localhost"
	port := ":8181"
	url := address + port + "/segment"

	filename := "hello-world.go"
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
