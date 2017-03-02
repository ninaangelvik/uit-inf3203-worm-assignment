package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const minx, maxx = 1, 3
const miny, maxy = 0, 59
const colwidth = 20
const gridLines = (maxx-minx+1) * ((maxy-miny) / colwidth + 2)
const refreshRate = 100 * time.Millisecond

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

	// Catch interrupt and quit
	interrupt := make(chan os.Signal, 2)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		fmt.Print(ansi_down_lines(gridLines))
		fmt.Println()
		log.Print("Shutting down")
		os.Exit(0)
	}()

	// Loop display forever
	for {
		nodeGrid(&statuses)
		time.Sleep(refreshRate)
	}
}

const ansi_bold = "\033[1m"
const ansi_reset = "\033[0m"
const ansi_reverse = "\033[30;47m"

func ansi_down_lines(n int) string {
	return fmt.Sprintf("\033[%dE", n)
}
func ansi_up_lines(n int) string {
	return fmt.Sprintf("\033[%dF", n)
}

func nodeGrid(statuses *map[string]status) {
	for x := minx; x <= maxx; x++ {
		for y := miny; y <= maxy; y++ {
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
	fmt.Print(ansi_up_lines(gridLines))
}
