package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type status struct {
	wormgate bool
	segment  bool
}

func main() {
	cmdline := []string{"/share/apps/bin/available-nodes.sh"}
	log.Printf("Getting list of nodes: %q", cmdline)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	out, err := cmd.Output()
	if err != nil {
		log.Panic("Error getting available nodes", err)
	}

	nodes := strings.Split(string(out), "\n")

	var statuses = make(map[string]status)
	for _, node := range nodes {
		statuses[node] = status{false, false}
	}

	nodeGrid(&statuses)
}

const ansi_bold = "\033[1m"
const ansi_reset = "\033[0m"
const ansi_reverse = "\033[30;47m"

func nodeGrid(statuses *map[string]status) {
	colwidth := 20
	for x := 1; x <= 3; x++ {
		for y := 0; y < 60; y++ {
			if y%colwidth == 0 {
				fmt.Printf("\n%d: %02d+", x, y/colwidth*colwidth)
			}
			if y%10 == 0 {
				fmt.Printf("|")
			}
			node := fmt.Sprintf("compute-%d-%d", x, y)
			status, nodeup := (*statuses)[node]

			var char string
			if nodeup {
				char = fmt.Sprint(y % 10)
			} else {
				char = " "
			}

			if status.wormgate {
				fmt.Print(ansi_bold)
			}
			if status.segment {
				fmt.Print(ansi_reverse)
			}
			fmt.Print(char)
			fmt.Print(ansi_reset)
		}
		fmt.Println()
	}
}
