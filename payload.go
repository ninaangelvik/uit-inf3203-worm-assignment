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
		log.Panic("Could not run QML man", err)
	}
}

func run() error {

        dir, _ := os.Getwd()

	print (dir)

	rand.Seed(time.Now().Unix())

	time.Sleep(5000 * time.Millisecond)

	return nil
}
