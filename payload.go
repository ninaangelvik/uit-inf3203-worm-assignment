package main

import (
	"log"
	"math/rand"
        "os"
	"time"
)

func main() {

	err := run()
	if err != nil {
		log.Panicf("Run error: %q", err)
	}
}

func run() error {

        dir, _ := os.Getwd()

	log.Printf("Payload app here! Running in %q", dir)

	rand.Seed(time.Now().Unix())

	time.Sleep(5000 * time.Millisecond)

	return nil
}
