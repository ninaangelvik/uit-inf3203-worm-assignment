package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"sync"
	"math/rand"
	"time"
)

var maxRunTime time.Duration

var wormgatePort string
var segmentPort string
var hostname string
var targetSegments int32
var killRateGuess float32

var segmentClient *http.Client

var segmentList struct {
	sync.RWMutex
	list []string 
}

type Result struct {
	host string
	ts int32
}

func createClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{},
	}
}

func main() {

	hostname, _ = os.Hostname()
	hostname = strings.Split(hostname, ".")[0]
	log.SetPrefix(hostname + " segment: ")

	var spreadMode = flag.NewFlagSet("spread", flag.ExitOnError)
	addCommonFlags(spreadMode)
	var spreadHost = spreadMode.String("host", "localhost", "host to spread to")

	var runMode = flag.NewFlagSet("run", flag.ExitOnError)
	addCommonFlags(runMode)

	if len(os.Args) == 1 {
		log.Fatalf("No mode specified\n")
	}

	segmentClient = createClient()


	killRateGuess = 0.0

	switch os.Args[1] {
	case "spread":
		spreadMode.Parse(os.Args[2:])
		sendSegment(*spreadHost)
	case "run":
		runMode.Parse(os.Args[2:])
		startSegmentServer()

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}

func addCommonFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&wormgatePort, "wp", ":8181", "wormgate port (prefix with colon)")
	flagset.StringVar(&segmentPort, "sp", ":8182", "segment port (prefix with colon)")
	flagset.DurationVar(&maxRunTime, "maxrun", time.Minute*10, "max time to run (in case you forget to shut down)")
}


func sendSegment(address string) bool {

	url := fmt.Sprintf("http://%s%s/wormgate?sp=%s", address, wormgatePort, segmentPort)
	filename := "tmp.tar.gz"

	log.Printf("Spreading to %s", url)

	// ship the binary and the qml file that describes our screen output
	tarCmd := exec.Command("tar", "-zc", "-f", filename, "segment")
	tarCmd.Run()
	defer os.Remove(filename)

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Could not read input file", err)
	}

	resp, err := http.Post(url, "string", file)
	if err != nil {
		log.Printf("POST error ", err)
		return false
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Println("Received OK from server")
	} else {
		return false
	}
	return true
}

func startSegmentServer() {
	// Quit if maxRunTime timeout
	exitReason := make(chan string, 1)
	go func() {
		time.Sleep(maxRunTime)
		exitReason <- fmt.Sprintf("maxrun timeout: %s", maxRunTime)
	}()
	go func() {
		reason := <-exitReason
		log.Printf(reason)
		log.Print("Shutting down")
		os.Exit(0)
	}()

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/targetsegments", targetSegmentsHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/shutdown_sibling", shutdownSiblingHandler)
	http.HandleFunc("/update_target", updateTargetSegmentHandler)
	http.HandleFunc("/get_target", getTargetSegmentsHandler)
	
	go getActiveSegments()
	go checkState()

	err := http.ListenAndServe(segmentPort, nil)
	if err != nil {
		log.Panic(err)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// We don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	fmt.Fprintf(w, "%.3f\n", killRateGuess)	
}


func targetSegmentsHandler(w http.ResponseWriter, r *http.Request) {

	var ts int32
	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	atomic.StoreInt32(&targetSegments, ts)

	updateTargetSegment(targetSegments)
	alterSegmentNumber()
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {

	var wg sync.WaitGroup
	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	for _,node := range(segmentList.list) {
		if node == hostname {
			continue
		}
		wg.Add(1)
		go httpGetShutdown(node, &wg)
	}
	wg.Wait()
	// Shut down
	log.Printf("Received shutdown command, committing suicide")
	os.Exit(0)
}

func shutdownSiblingHandler(w http.ResponseWriter, r *http.Request) {

	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// Shut down
	log.Printf("Received shutdown command, committing suicide")
	os.Exit(0)
}

func getTargetSegmentsHandler(w http.ResponseWriter, r *http.Request) {
	// We don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	fmt.Fprintf(w, "%d", targetSegments)
}


func fetchReachableHosts() []string {
	url := fmt.Sprintf("http://localhost%s/reachablehosts", wormgatePort)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("error fetching hosts: %s", err)
		return []string{}
	}

	var bytes []byte
	bytes, err = ioutil.ReadAll(resp.Body)
	body := string(bytes)
	resp.Body.Close()

	trimmed := strings.TrimSpace(body)
	nodes := strings.Split(trimmed, "\n")

	return nodes
}

func shutdownSibling(address string) {
	url := fmt.Sprintf("http://%s%s/shutdown_sibling", address, segmentPort)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error shutting down %s", address)
	}
	if err == nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func getActiveSegments() {
	for {
		updateSegmentList()
		time.Sleep(1 * time.Second)
	}
}

func checkState() {
	if targetSegments > 0 && targetSegments != int32(len(segmentList.list)) {
		alterSegmentNumber()
	}
	time.Sleep(2 * time.Second)
}

func updateSegmentList(){
	var wg sync.WaitGroup
	segmentChannel := make(chan Result, 84)
	var activeSegments []string

	reachableHosts := fetchReachableHosts()

	if len(reachableHosts) > 0 {	
		for _,host := range(reachableHosts) {
			wg.Add(1)
			go httpGetTargetSegment(segmentChannel, host, &wg)
		}
		wg.Wait()
		close(segmentChannel)

		var result Result
		if len(segmentChannel) > 0 {
			length := len(segmentChannel)
			for i := 0; i < length; i++ {
				result = <-segmentChannel
				activeSegments = append(activeSegments, result.host)

				if targetSegments == 0 && result.ts != 0{
					atomic.StoreInt32(&targetSegments, result.ts)
				}
			}

			segmentList.Lock()
			segmentList.list = activeSegments
			segmentList.Unlock()
		}
	}

	if targetSegments > 0 && targetSegments != int32(len(segmentList.list)) {
		alterSegmentNumber()
	}
}

func httpGetTargetSegment(c chan Result, host string, wg *sync.WaitGroup) {
	var ts int32
	var result Result
	url := fmt.Sprintf("http://%s%s/get_target", host, segmentPort)
	resp, err := segmentClient.Get(url)
	isOk := err == nil && resp.StatusCode == 200

	if err != nil {
		log.Printf("Error checking %s: %s", url, err)
	} else {
		_,_ = fmt.Fscanf(resp.Body, "%d", &ts)
		// Consume and close rest of body
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	if isOk == true {
		result.host = host	
		result.ts = ts
		c <- result
	}
	wg.Done()
	
}

func updateTargetSegment(newTarget int32) {
	var wg sync.WaitGroup

	for _,node := range(segmentList.list){
		wg.Add(1)
		go httpPostTargetSegment(node, newTarget, &wg)
	}

	wg.Wait()
}

func httpPostTargetSegment(host string, newTarget int32, wg *sync.WaitGroup) {
	url := fmt.Sprintf("http://%s%s/update_target", host, segmentPort)
	postBody := strings.NewReader(fmt.Sprint(newTarget))

	resp, err := segmentClient.Post(url, "text/plain", postBody)
	if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
		log.Printf("Error updating target segment")
	}
	if err == nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	wg.Done()		
}

func httpGetShutdown(address string, wg *sync.WaitGroup) {
	url := fmt.Sprintf("http://%s%s/shutdown", address, segmentPort)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error shutting down %s", address)
	}
	if err == nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
	wg.Done()
}


func updateTargetSegmentHandler(w http.ResponseWriter, r *http.Request) {
	pc, rateErr := fmt.Fscanf(r.Body, "%d", &targetSegments)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing nodes: %s", pc, rateErr)
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}

func alterSegmentNumber() {
	segmentList.RLock()

	if len(segmentList.list) == 0 {
		time.Sleep(10 * time.Millisecond)
	}

	if int32(len(segmentList.list)) < targetSegments {
		reachablehosts := fetchReachableHosts()
		new_host := reachablehosts[rand.Intn(len(reachablehosts))]
		sendSegment(new_host)
	}

	if int32(len(segmentList.list)) > targetSegments {
		shutdownHost := segmentList.list[rand.Intn(len(segmentList.list))]
		shutdownSibling(shutdownHost)
	}

  segmentList.RUnlock()
  time.Sleep(1 * time.Second)
}

