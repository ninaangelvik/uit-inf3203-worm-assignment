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

// var wormgates = []string {"compute-3-21", "compute-3-22", "compute-3-23", "compute-3-24", "compute-3-26", "compute-3-27", "compute-3-28"}

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


	log.Printf("Ending main")
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

	// updateSegmentList()
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
	http.HandleFunc("/update_target", updateTargetSegmentHandler)
	http.HandleFunc("/get_target", getTargetSegmentsHandler)
	
	go getActiveSegments()

	go checkState()

	// log.Printf("Before ListenAndServe")	
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
	log.Printf("Length of segmentlist: %d, ts: %d", len(segmentList.list), targetSegments)
	alterSegmentNumber()
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {

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

func shutdown(address string) {
	url := fmt.Sprintf("http://%s%s/shutdown", address, segmentPort)
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
	
	// var start time.Time
	// var elapsed time.Duration 
	// var killed int
	for {
		updateSegmentList()
		time.Sleep(1 * time.Second)
		// checkState()
	}
}

func checkState() {
	
	log.Printf("Len: %d, ts: %d", len(segmentList.list), targetSegments)
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
		// reachableHosts := wormgates
		// log.Printf("len reachablehosts %d", len(reachableHosts))
		if len(reachableHosts) > 0 {	
			for _,host := range(reachableHosts) {
				wg.Add(1)
				go httpGetTargetSegment(segmentChannel, host, &wg)
			}
			wg.Wait()
			close(segmentChannel)

			// log.Printf("Closing channel at %.3f seconds", time.Duration.Seconds(time.Since(start)))
			var result Result

			// result = <- segmentChannel
			// log.Printf("len  %d", len(segmentChannel))

			if len(segmentChannel) > 0 {
				length := len(segmentChannel)
				for i := 0; i < length; i++ {
					result = <-segmentChannel
					log.Printf("appending")
					activeSegments = append(activeSegments, result.host)

					if targetSegments == 0 && result.ts != 0{
						atomic.StoreInt32(&targetSegments, result.ts)
					}
				}

				// log.Printf("Locking list at %.3f", time.Duration.Seconds(time.Since(start)))
				segmentList.Lock()
				segmentList.list = activeSegments
				segmentList.Unlock()
				// log.Printf("List is updated %s", strings.Join(segmentList.list, ", "))
			}
		}

		if targetSegments > 0 && targetSegments != int32(len(segmentList.list)) {
			log.Printf("Len: %d, ts: %d", len(segmentList.list), targetSegments)
			alterSegmentNumber()
		}

		// log.Printf("exit")
}

func httpGetTargetSegment(c chan Result, host string, wg *sync.WaitGroup) {
	var ts int32
	var result Result
	url := fmt.Sprintf("http://%s%s/get_target", host, segmentPort)
	resp, err := segmentClient.Get(url)
	isOk := err == nil && resp.StatusCode == 200

	if err != nil {
			// log.Printf("Error checking %s: %s", url, err)
	} else {
		_,_ = fmt.Fscanf(resp.Body, "%d", &ts)
		// Consume and close rest of body
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	if isOk == true {
		// log.Printf("%s is ok", host)
		result.host = host	
		result.ts = ts
		c <- result
	}
	// log.Printf("Ending get request at %.4f", time.Duration.Seconds(time.Since(start)))
	wg.Done()
	
}

// func httpGetOk(c chan string, host string) {
// 	url := fmt.Sprintf("http://%s%s/", host, segmentPort)
// 	resp, err := segmentClient.Get(url)
// 	// isOk := err == nil && resp.StatusCode == 200

// 	if err != nil {
// 		if strings.Contains(fmt.Sprint(err), "connection refused") {
// 			// ignore connection refused errors
// 			err = nil
// 		} else {
// 			// log.Printf("Error checking %s: %s", url, err)
// 		}
// 	} else {
// 		resp.Body.Close()
// 	}
// }

func updateTargetSegment(newTarget int32) {
	var wg sync.WaitGroup

	log.Printf("Posting targetsegment to other nodes")
	for _,node := range(segmentList.list){
		wg.Add(1)
		go httpPostTargetSegment(node, newTarget, &wg)
	}

	wg.Wait()
	log.Printf("Done posting targetSegments")
	// time.Sleep(25 * time.Millisecond)
}

func httpPostTargetSegment(host string, newTarget int32, wg *sync.WaitGroup) {
	url := fmt.Sprintf("http://%s%s/update_target", host, segmentPort)
	postBody := strings.NewReader(fmt.Sprint(newTarget))

	resp, err := segmentClient.Post(url, "text/plain", postBody)
	if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
		log.Printf("Error updating target segment")
		// postRemoveNode(node)
	}
	if err == nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	wg.Done()		
}

func updateTargetSegmentHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Updating targetsegments at %s", hostname)

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
	log.Printf("enter alterSegmentNumber")

	if len(segmentList.list) == 0 {
		// log.Printf("Length of list: %d, ts: %d", len(segmentList.list), targetSegments)
		time.Sleep(10 * time.Millisecond)
	}

	if int32(len(segmentList.list)) < targetSegments {
		log.Printf("adding segment")
		reachablehosts := fetchReachableHosts()
		// reachablehosts := wormgates
		new_host := reachablehosts[rand.Intn(len(reachablehosts))]
		
		sendSegment(new_host)
		// for retval == false {
		// 	new_host = reachablehosts[rand.Intn(len(reachablehosts))]
		// 	retval = sendSegment(new_host)
		// }
		// updateTargetSegment(targetSegments)
	}
	if int32(len(segmentList.list)) > targetSegments {
		log.Printf("removing segment")
		shutdownHost := segmentList.list[rand.Intn(len(segmentList.list))]
		// for shutdownHost == hostname {
		// 	shutdownHost = segmentList.list[rand.Intn(len(segmentList.list))]
		// }
		shutdown(shutdownHost)
	}
  segmentList.RUnlock()
  time.Sleep(1 * time.Second)
	log.Printf("exit alterSegmentNumber")
}

