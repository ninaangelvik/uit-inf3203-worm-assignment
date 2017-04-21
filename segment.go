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
	// "math/rand"
	"time"
)

var wormgatePort string
var segmentPort string
var hostname string
var targetSegments int32
var	nodeList []string 

var segmentClient *http.Client

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


func sendSegment(address string) int {

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
		return 0
	} else {
		log.Println("Response: ", resp)
		return -1
	}

	log.Printf("HOOOOOOOOOOST: %s", hostname)
	return 1
	// nodeList = append(nodeList, address)

}

func startSegmentServer() {
	// log.Printf("In startSegmentServer at host: %s", hostname)

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/targetsegments", targetSegmentsHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	// http.HandleFunc("/share_nodes", shareNeighborHandler)
	// http.HandleFunc("/list_neighbors", listNeighborHandler)
	// http.HandleFunc("/add_node", addNodeHandler)
	// http.HandleFunc("/remove_node", removeNodeHandler)
	// http.HandleFunc("/update_target", updateTargetSegmentHandler)

	// log.Printf("Starting segment server on %s%s\n", hostname, segmentPort)
	// nodeList = append(nodeList, hostname)
	// log.Printf("Neighbors in start: %s", strings.Join(nodeList," "))
	// go scheduler(ticker)
	
	go getActiveSegments()
	// log.Printf("Reachable hosts: %s", strings.Join(fetchReachableHosts()," "))
	err := http.ListenAndServe(segmentPort, nil)
	if err != nil {
		log.Panic(err)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// We don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	killRateGuess := 2.0

	
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
	log.Printf("Nodes in list: %s", strings.Join(nodeList, ", "))
	// log.Printf("%d", targetSegments)
	// updateTargetSegment(targetSegments)
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {

	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// Shut down
	log.Printf("Received shutdown command, committing suicide")
	os.Exit(0)
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
	var wg sync.WaitGroup
	segmentChannel := make(chan string, 84)

	for {
		var activeSegments []string
		log.Printf("in getActiveSegments")
		reachableHosts := fetchReachableHosts()

		for _,host := range(reachableHosts) {
			wg.Add(1)
			go httpGetOk(segmentChannel, host, &wg)
		}
		wg.Wait()

		for i := 0; i < len(segmentChannel); i++ {
			activeSegments = append(activeSegments, <-segmentChannel)
		}

		nodeList = activeSegments

		time.Sleep(25 * time.Millisecond)
	}
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
			log.Printf("Error checking %s: %s", url, err)
		}
	} else {
		resp.Body.Close()
	}

	if isOk == true {
		log.Printf("IN httpGetOk: host = %s", host)
		c <- host
	}
	
	wg.Done()
}


// func updateTargetSegment(newTarget int32) {
// 	log.Printf("HOSTNAME: %s", hostname)
	
// 	for _,node := range(nodeList){
// 		url := fmt.Sprintf("http://%s%s/update_target", node, segmentPort)
// 		postBody := strings.NewReader(fmt.Sprint(newTarget))

// 		resp, err := http.Post(url, "text/plain", postBody)
// 		if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
// 			log.Printf("Error updating target segment")
// 			postRemoveNode(node)
// 		}
// 		if err == nil {
// 			io.Copy(ioutil.Discard, resp.Body)
// 			resp.Body.Close()
// 		}		
// 	}

// 	shutdown(node)
// 	if int32(len(nodeList)) != targetSegments {
// 		checkState()
// 	}
// }

// func httpPostTargetSegment(client *http.Client, url string) {

// }
// func updateTargetSegmentHandler(w http.ResponseWriter, r *http.Request) {
// 	log.Printf("Updating targetsegments at %s", hostname)

// 	pc, rateErr := fmt.Fscanf(r.Body, "%d", &targetSegments)
// 	if pc != 1 || rateErr != nil {
// 		log.Printf("Error parsing nodes: %s", pc, rateErr)
// 	}

// 	// Consume and close rest of body
// 	io.Copy(ioutil.Discard, r.Body)
// 	r.Body.Close()

// 	log.Printf("&&&&&&&& Host: %s , Targetsegment: %d &&&&&&&6", hostname, targetSegments)
// }


// func shareNeighborHandler(w http.ResponseWriter, r *http.Request) {
// 	log.Printf("?????????????????????????????????????????")
// 	var bytes []byte
// 	bytes,_ = ioutil.ReadAll(r.Body)
// 	body := string(bytes)
// 	r.Body.Close()
// 	log.Printf("OLD LIST: %s", strings.Join(nodeList, ", "))
// 	trimmed := strings.TrimSpace(body)
// 	nodes := strings.Split(trimmed, " ")
// 	nodeList = append(nodeList, nodes...)
// 	log.Printf("NEW LIST: %s", strings.Join(nodeList, ", "))


// }

// func addNodeHandler(w http.ResponseWriter, r *http.Request) {
// 	var node string
// 	pc, rateErr := fmt.Fscanf(r.Body, "%s", &node)
// 	if pc != 1 || rateErr != nil {
// 		log.Printf("Error parsing nodes: %s", pc, rateErr)
// 	}

// 	// Consume and close rest of body
// 	io.Copy(ioutil.Discard, r.Body)
// 	r.Body.Close()

// 	nodeList = append(nodeList, node)
// }

// func removeNodeHandler(w http.ResponseWriter, r *http.Request) {
// 	var node string
// 	pc, rateErr := fmt.Fscanf(r.Body, "%s", &node)
// 	if pc != 1 || rateErr != nil {
// 		log.Printf("Error parsing nodes: %s", pc, rateErr)
// 	}

// 	// Consume and close rest of body
// 	io.Copy(ioutil.Discard, r.Body)
// 	r.Body.Close()

// 	nodeList = remove(nodeList, node)
// }


// func checkState() {
// 	// log.Printf("------- HOST: %s targetsegments: %d-----------", hostname, targetSegments)

// 	log.Printf("In checkState")
	
// 	for int32(len(nodeList)) < targetSegments {
// 		reachablehosts := fetchReachableHosts()
// 		new_host := reachablehosts[rand.Intn(len(reachablehosts))]
// 		for new_host == hostname {
// 			reachablehosts = fetchReachableHosts()
// 			new_host = reachablehosts[rand.Intn(len(reachablehosts))]
// 		}

// 		log.Printf(new_host)
// 		log.Printf(wormgatePort)
// 		log.Printf(segmentPort)
// 		retval := sendSegment(new_host)
// 		if retval == 0 {
// 			postAddNode(new_host)
// 			log.Printf("Neighbors: %s", strings.Join(nodeList," "))
// 			log.Printf("Neighbors: %s", strings.Join(nodeList," "))
// 			time.Sleep(time.Second)
// 			shareNeighbors(new_host)
// 			nodeList = append(nodeList, new_host)
// 		} else {
// 			log.Printf("Error in checkState")
// 		}
// 	}
// 	for int32(len(nodeList)) > targetSegments {
// 		log.Printf("%d, %d", len(nodeList), targetSegments)
// 		log.Printf("Too many in list")
// 		node := hostname
// 		postRemoveNode(node)
// 		shutdown(node)
// 		nodeList = remove(nodeList, node)
// 	}
// }

// func postAddNode(address string) {
// 	for _,node := range(nodeList) {
// 		if node == hostname {
// 			continue
// 		}
// 		log.Printf("NODE: %s", node)
// 		url := fmt.Sprintf("http://%s%s/add_node", node, segmentPort)
// 		postBody := strings.NewReader(fmt.Sprint(address))

// 		resp, err := http.Post(url, "text/plain", postBody)
// 		if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
// 			log.Printf("Error adding node to %s: %s", node, err)
// 		}
// 		if err == nil {
// 			io.Copy(ioutil.Discard, resp.Body)
// 			resp.Body.Close()
// 		}
// 	}
// }

// func postRemoveNode(address string) {
// 	for _,node := range(nodeList) {
// 		if node == hostname {
// 			continue
// 		}
// 		url := fmt.Sprintf("http://%s%s/remove_node", node, segmentPort)
// 		postBody := strings.NewReader(fmt.Sprint(address))

// 		resp, err := http.Post(url, "text/plain", postBody)
// 		if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
// 			log.Printf("Error adding node to to %s: %s", node, err)
// 		}
// 		if err == nil {
// 			io.Copy(ioutil.Discard, resp.Body)
// 			resp.Body.Close()
// 		}
// 	}
// }

// func shareNeighbors(address string) {
// 	url := fmt.Sprintf("http://%s%s/share_nodes", address, segmentPort)

// 	log.Printf("Sending neighbors %s to %s", strings.Join(nodeList," "),url)

// 	// ship the binary and the qml file that describes our screen output
// 	postBody :=  strings.NewReader(fmt.Sprint(strings.Join(nodeList," ")))
// 	resp, err := http.Post(url, "text/plain", postBody)
// 	if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
// 		log.Printf("Error sharing neighbors with %s: %s", address, err)
// 	}
// 	if err == nil {
// 		io.Copy(ioutil.Discard, resp.Body)
// 		resp.Body.Close()
// 	}
// }

// func remove(s []string, r string) []string {
//     for i, v := range s {
//         if v == r {
//             return append(s[:i], s[i+1:]...)
//         }
//     }
//     return s
// }
