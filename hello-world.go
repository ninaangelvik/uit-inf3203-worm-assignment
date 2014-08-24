package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	// Open output file. Create if not found, read-write and append to it.
	filename := "output"
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Panic("Could not open file", err)
	}

	// Write something 10 times, sleep inbetween writes.
	for i := 0; i < 10; i++ {
		iteration := strconv.Itoa(i)
		str := "Hello world. Iteration no. " + iteration + "\n"
		_, err = file.Write([]byte(str))
		if err != nil {
			log.Panic("Could not write to file", err)
		}
		time.Sleep(100 * time.Millisecond)

	}

}
