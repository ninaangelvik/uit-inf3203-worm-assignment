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
	"math/rand"
)

var wormgatePort string
var segmentPort string
var hostname string
var targetSegments int32
var	nodeList []string 

func main() {

	hostname, _ = os.Hostname()
	log.SetPrefix(hostname + " segment: ")

	var spreadMode = flag.NewFlagSet("spread", flag.ExitOnError)
	addCommonFlags(spreadMode)
	var spreadHost = spreadMode.String("host", "localhost", "host to spread to")

	var runMode = flag.NewFlagSet("run", flag.ExitOnError)
	addCommonFlags(runMode)

	if len(os.Args) == 1 {
		log.Fatalf("No mode specified\n")
	}

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
	log.Printf("In startSegmentServer")

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/targetsegments", targetSegmentsHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/share_neighbors", shareNeighborHandler)
	http.HandleFunc("/list_neighbors", listNeighborHandler)
	http.HandleFunc("/add_node", addNodeHandler)
	http.HandleFunc("/remove_node", removeNodeHandler)
	http.HandleFunc("/update_target", updateTargetSegmentHandler)

	log.Printf("Starting segment server on %s%s\n", hostname, segmentPort)
	nodeList = append(nodeList, (strings.Split(hostname, ".")[0]))
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

	// log.Printf("Length of list: %d", len(nodeList))
	// log.Printf("Neighbors: %s", strings.Join(nodeList," "))
	// if int32(len(nodeList)) != targetSegments {
	// 	checkState()
	// }

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
	log.Printf("%d", targetSegments)
	updateTargetSegment(targetSegments)
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {

	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// Shut down
	log.Printf("Received shutdown command, committing suicide")
	os.Exit(0)
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

	log.Printf("&&&&&&&& Host: %s , Targetsegment: %d &&&&&&&6", hostname, targetSegments)
}


func listNeighborHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("send list of neighbors")
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
	log.Printf("+++++  %s, Neighbors: %s  ++++++++++", hostname, strings.Join(nodeList," "))

	for _,host := range(nodeList) {
		fmt.Fprintln(w, host)
	}
}

func shareNeighborHandler(w http.ResponseWriter, r *http.Request) {
	var bytes []byte
	bytes,_ = ioutil.ReadAll(r.Body)
	body := string(bytes)
	r.Body.Close()
	log.Printf("OLD LIST: %s", strings.Join(nodeList, ", "))
	trimmed := strings.TrimSpace(body)
	nodes := strings.Split(trimmed, " ")
	nodeList = append(nodeList, nodes...)
	log.Printf("NEW LIST: %s", strings.Join(nodeList, ", "))


}

func addNodeHandler(w http.ResponseWriter, r *http.Request) {
	var node string
	pc, rateErr := fmt.Fscanf(r.Body, "%s", &node)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing nodes: %s", pc, rateErr)
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	nodeList = append(nodeList, node)
}

func removeNodeHandler(w http.ResponseWriter, r *http.Request) {
	var node string
	pc, rateErr := fmt.Fscanf(r.Body, "%s", &node)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing nodes: %s", pc, rateErr)
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	nodeList = remove(nodeList, node)
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

func updateTargetSegment(newTarget int32) {
	log.Printf("HOSTNAME: %s", hostname)
	
	for _,node := range(nodeList){
		url := fmt.Sprintf("http://%s%s/update_target", node, segmentPort)
		postBody := strings.NewReader(fmt.Sprint(newTarget))

		resp, err := http.Post(url, "text/plain", postBody)
		if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
			log.Printf("Error updating target segment")
			postRemoveNode(node)
			shutdown(node)
		}
		if err == nil {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}		
	}

	if int32(len(nodeList)) != targetSegments {
		checkState()
	}
}

func checkState() {
	// log.Printf("------- HOST: %s targetsegments: %d-----------", hostname, targetSegments)

	log.Printf("Length of list: %d", len(nodeList))
	log.Printf("Neighbors: %s", strings.Join(nodeList," "))
	
	for int32(len(nodeList)) < targetSegments {
		reachablehosts := fetchReachableHosts()
		new_host := reachablehosts[rand.Intn(len(reachablehosts))]
		for new_host == strings.Split(hostname, ".")[0] {
			reachablehosts = fetchReachableHosts()
			new_host = reachablehosts[rand.Intn(len(reachablehosts))]
		}

		log.Printf(new_host)
		log.Printf(wormgatePort)
		log.Printf(segmentPort)
		retval := sendSegment(new_host)
		if retval == 0 {
			postAddNode(new_host)
			nodeList = append(nodeList, new_host)
			shareNeighbors(new_host)
		}
	}
	for int32(len(nodeList)) > targetSegments {
		node := strings.Split(hostname, ".")[0]
		postRemoveNode(node)
		shutdown(node)
	}
}

func postAddNode(address string) {
	for _,node := range(nodeList) {
		log.Printf("NODE: %s", node)
		url := fmt.Sprintf("http://%s%s/add_node", node, segmentPort)
		postBody := strings.NewReader(fmt.Sprint(address))

		resp, err := http.Post(url, "text/plain", postBody)
		if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
			log.Printf("Error adding node to %s: %s", node, err)
		}
		if err == nil {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}
	}
}

func postRemoveNode(address string) {
	for _,node := range(nodeList) {
		url := fmt.Sprintf("http://%s%s/remove_node", node, segmentPort)
		postBody := strings.NewReader(fmt.Sprint(address))

		resp, err := http.Post(url, "text/plain", postBody)
		if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
			log.Printf("Error adding node to to %s: %s", node, err)
		}
		if err == nil {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}
	}
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

func shareNeighbors(address string) {
	url := fmt.Sprintf("http://%s%s/share_neighbors", address, segmentPort)

	log.Printf("Sending neighbors to %s", url)

	// ship the binary and the qml file that describes our screen output
	postBody :=  strings.NewReader(strings.Join(nodeList," "))
	resp, err := http.Post(url, "text/plain", postBody)
	if err != nil && !strings.Contains(fmt.Sprint(err), "refused") {
		log.Printf("Error sharing neighbors with %s: %s", address, err)
	}
	if err == nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func remove(s []string, r string) []string {
    for i, v := range s {
        if v == r {
            return append(s[:i], s[i+1:]...)
        }
    }
    return s
}