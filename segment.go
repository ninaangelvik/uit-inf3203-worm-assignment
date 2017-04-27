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
	// "time"
)

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

	updateSegmentList()

	log.Printf("Length of list: %d", len(segmentList.list))
	if len(segmentList.list) == 1 {
		targetSegments = 1
	} else {
		for _, host := range(segmentList.list) {
			httpGetTargetSegment(host)
			if targetSegments > 1 {
				break
			} 
		}
	}

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
		log.Panic("Could not read input file", err)
	}

	resp, err := http.Post(url, "string", file)
	if err != nil {
		log.Panic("POST error ", err)
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
	// log.Printf("In startSegmentServer at host: %s", hostname)

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/targetsegments", targetSegmentsHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/update_target", updateTargetSegmentHandler)
	http.HandleFunc("/get_target", getTargetSegmentsHandler)
	
	go getActiveSegments()

	log.Printf("TargetSegments: %d", targetSegments)
	// go checkState()

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

func httpGetTargetSegment(host string) {
	var ts int32
	url := fmt.Sprintf("http://%s%s/get_segment", host, segmentPort)
	resp, err := segmentClient.Get(url)

	if err != nil {
			// log.Printf("Error checking %s: %s", url, err)
	} else {
		_,_ = fmt.Fscanf(resp.Body, "%d", &ts)
		// Consume and close rest of body
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()

		atomic.StoreInt32(&targetSegments, ts)
	}
}

func fetchReachableHosts() []string {
	url := fmt.Sprintf("http://localhost%s/reachablehosts", wormgatePort)
	resp, err := http.Get(url)
	if err != nil {
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

		if targetSegments > 1{
			alterSegmentNumber()
		}

		// time.Sleep(10 * time.Millisecond)
	}
}

func updateSegmentList(){
		var wg sync.WaitGroup
		segmentChannel := make(chan string, 84)
		var activeSegments []string

		reachableHosts := fetchReachableHosts()

		for _,host := range(reachableHosts) {
			wg.Add(1)
			go httpGetOk(segmentChannel, host, &wg)
		}
		wg.Wait()
		close(segmentChannel)

		for i := 0; i <= len(segmentChannel); i++ {
			segment := <-segmentChannel
			activeSegments = append(activeSegments, segment)
		}

		segmentList.Lock()
		segmentList.list = activeSegments
		segmentList.Unlock()

}

func httpGetOk(c chan string, host string, wg *sync.WaitGroup) {
	url := fmt.Sprintf("http://%s%s/", host, segmentPort)
	resp, err := segmentClient.Get(url)
	isOk := err == nil && resp.StatusCode == 200

	if err != nil {
		if strings.Contains(fmt.Sprint(err), "connection refused") {
			// ignore connection refused errors
			err = nil
		} else {
			// log.Printf("Error checking %s: %s", url, err)
		}
	} else {
		resp.Body.Close()
	}

	if isOk == true {
		c <- host
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
	// log.Printf("LOCKED in alterSegmentNumber")

	// if targetSegments != int32(len(segmentList.list)) {
		if int32(len(segmentList.list)) < targetSegments {
			reachablehosts := fetchReachableHosts()
			new_host := reachablehosts[rand.Intn(len(reachablehosts))]
			

			retval := sendSegment(new_host)
			for retval == false {
				new_host = reachablehosts[rand.Intn(len(reachablehosts))]
				retval = sendSegment(new_host)
			}
			// updateTargetSegment(targetSegments)
		}
		if int32(len(segmentList.list)) > targetSegments {
			shutdownHost := segmentList.list[rand.Intn(len(segmentList.list))]
			for shutdownHost == hostname {
				shutdownHost = segmentList.list[rand.Intn(len(segmentList.list))]
			}
			shutdown(shutdownHost)
		}
  // }
  segmentList.RUnlock()
	// log.Printf("UNLOCKED in alterSegmentNumber")
}


// func checkState() {
// 	// log.Printf("LOCKED in alterSegmentNumber")
// 	for {
// 		// segmentList.RLock()
// 		if (targetSegments != int32(len(segmentList.list))) && (targetSegments > 1){
// 			if int32(len(segmentList.list)) < targetSegments {
// 				// log.Printf("ADD NODE")
// 				reachablehosts := fetchReachableHosts()
// 				new_host := reachablehosts[rand.Intn(len(reachablehosts))]
			
// 				retval := sendSegment(new_host)
// 				for retval == false {
// 					new_host = reachablehosts[rand.Intn(len(reachablehosts))]
// 					retval = sendSegment(new_host)
// 				}
// 			}
// 			if int32(len(segmentList.list)) > targetSegments {
// 				// log.Printf("Remove NODE")
// 				shutdownHost := segmentList.list[rand.Intn(len(segmentList.list))]
// 				for shutdownHost == hostname {
// 					shutdownHost = segmentList.list[rand.Intn(len(segmentList.list))]
// 				}
// 				shutdown(shutdownHost)
// 			}
// 	  }
//   	// segmentList.RUnlock()
// 		time.Sleep(2 * time.Second)
//   }
// 	// log.Printf("UNLOCKED in alterSegmentNumber")
// }