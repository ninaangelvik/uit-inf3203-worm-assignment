package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const minx, maxx = 1, 3
const miny, maxy = 0, 59
const colwidth = 20
const gridLines = (maxx - minx + 1) * ((maxy-miny)/colwidth + 2) + 4
const refreshRate = 100 * time.Millisecond
const pollRate = refreshRate / 2

const wormgatePort = ":8181"
const segmentPort = ":8182"

type status struct {
	wormgate bool
	segment  bool
}

type statusMap struct {
	sync.RWMutex
	m map[string]status
}

var killRate struct {
	sync.RWMutex
	r int
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

	var statuses = statusMap{m: make(map[string]status)}
	for _, node := range nodes {
		statuses.m[node] = status{false, false}
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

	// Start poll routines
	for node, _ := range statuses.m {
		go pollNodeForever(&statuses, node)
	}

	// Start input routine
	go inputHandler()

	// Loop display forever
	for {
		nodeGrid(&statuses)
		time.Sleep(refreshRate)
	}
}

func pollNodeForever(statuses *statusMap, node string) {
	log.Printf("Starting poll routine for %s", node)
	for {
		s := pollNode(node)
		statuses.Lock()
		statuses.m[node] = s
		statuses.Unlock()
		time.Sleep(pollRate)
	}
}

func pollNode(host string) status {
	wormgateUrl := fmt.Sprintf("http://%s%s/", host, wormgatePort)
	segmentUrl := fmt.Sprintf("http://%s%s/", host, segmentPort)
	wormgate := httpGetOk(wormgateUrl)
	segment := httpGetOk(segmentUrl)

	return status{wormgate, segment}
}

func httpGetOk(url string) bool {
	resp, err := http.Get(url)
	ok := err==nil && resp.StatusCode == 200
	if err==nil {
		resp.Body.Close()
	}
	return ok
}

func inputHandler() {
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		killRate.Lock()
		for _, ch := range input {
			switch ch {
			case 'u':
				killRate.r += 1
			case 'U':
				killRate.r += 10
			case 'd':
				killRate.r -= 1
			case 'D':
				killRate.r -= 10
			}
		}
		if killRate.r < 0 {
			killRate.r = 0
		}
		killRate.Unlock()
	}
}

const ansi_bold = "\033[1m"
const ansi_reset = "\033[0m"
const ansi_reverse = "\033[30;47m"
const ansi_clear_to_end = "\033[0J"

func ansi_down_lines(n int) string {
	return fmt.Sprintf("\033[%dE", n)
}
func ansi_up_lines(n int) string {
	return fmt.Sprintf("\033[%dF", n)
}

func nodeGrid(statuses *statusMap) {
	statuses.RLock()
	defer statuses.RUnlock()

	fmt.Print(ansi_clear_to_end)
	fmt.Println()
	for x := minx; x <= maxx; x++ {
		for y := miny; y <= maxy; y++ {
			if y%colwidth == 0 {
				fmt.Printf("\n%d: %02d+", x, y/colwidth*colwidth)
			}
			if y%10 == 0 {
				fmt.Printf("|")
			}
			node := fmt.Sprintf("compute-%d-%d", x, y)
			status, nodeup := statuses.m[node]

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
	fmt.Println()

	killRate.RLock()
	fmt.Printf("Kill rate: %d/sec\n", killRate.r)
	killRate.RUnlock()

	fmt.Println(time.Now().Format(time.StampMilli))
	fmt.Print(ansi_up_lines(gridLines))
}
