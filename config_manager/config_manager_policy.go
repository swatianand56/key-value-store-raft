package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// command line arguments
type ConfigMgrArgs struct {
	onlyOnce bool
	hurry    bool
}

func isElementPresentInArray(arr []int, ele int) bool {
	for _, i := range arr {
		if i == ele {
			return true
		}
	}
	return false
}

func parseArgs(args []string) ConfigMgrArgs {
	arguments := ConfigMgrArgs{}

	for _, arg := range args {
		switch arg {
		case "--once":
			arguments.onlyOnce = true
		case "--hurry":
			arguments.hurry = true
		}
	}

	return arguments
}

func main() {
	args := parseArgs(os.Args[1:])

	serverList := []string{
		"localhost:8001",
		"localhost:8002",
		"localhost:8003",
		"localhost:8004",
		"localhost:8005",
		"localhost:8006",
		"localhost:8007",
		"localhost:8008",
		"localhost:8009",
		"localhost:8010",
		"localhost:8011",
	}
	activeServers := []int{0, 1, 2}
	unreachableServers := []int{}
	failureTestTime := 10 // time in seconds
	failureHandleCapacity := 1
	activeServersReqd := 3
	const FailureCycles = 3
	numFailureCycles := FailureCycles
	alive := true

	for alive == true {
		if args.onlyOnce == true {
			alive = false
		}
		if args.hurry == false {
			fmt.Println("sleeping for 10 seconds")
			time.Sleep(time.Duration(failureTestTime) * time.Second)
		}

		// start with 3 machines: least number, can handle the failure of 1 machine (if a machine goes down -- up another machine)
		// if n machines have failed in the last m minutes, increase failure tolerance by n+1 (max machines in our case = 11 -- this can tolerate the failure of 5 machines)
		// if 0 machines have failed in the last m minutes, decrease failure tolerance by 1 (min number of machines being 3)
		// ping all the active servers

		fmt.Println("active servers are ---- ", activeServers, failureHandleCapacity, unreachableServers)

		var serverStatus int
		numFailures := 0

		// make a check over unreachable servers and remove them from the list if they are now reachable

		for index, server := range unreachableServers {
			fmt.Println("checking unreachable server ", server)
			if kv739_init([]string{serverList[server]}, 1) == 0 {
				unreachableServers[index] = -1
			}
		}

		sort.Slice(unreachableServers, func(i, j int) bool {
			return unreachableServers[i] < unreachableServers[j]
		})

		for index, server := range unreachableServers {
			if server == -1 {
				unreachableServers = unreachableServers[:index]
				break
			}
		}

		for index, server := range activeServers {
			fmt.Println("checking active server ", server)
			serverStatus = kv739_init([]string{serverList[server]}, 1)
			if serverStatus == -1 { // error connecting
				unreachableServers = append(unreachableServers, server)
				activeServers[index] = activeServers[len(activeServers)-1]
				activeServers = activeServers[:len(activeServers)-1]
				numFailures++
				fmt.Println("failure encountered new active servers are ", activeServers, unreachableServers)
			}
		}
		if numFailures == 0 {
			if numFailureCycles == 0 {
				failureHandleCapacity = int(math.Max(float64(failureHandleCapacity-1), float64(1)))
				numFailureCycles = FailureCycles
			} else {
				numFailureCycles--
			}
		} else {
			failureHandleCapacity = failureHandleCapacity + numFailures
		}
		activeServersReqd = (2 * failureHandleCapacity) + 1

		fmt.Println("num failures, failure handle capacity, active servers required ", numFailures, failureHandleCapacity, activeServersReqd)

		if activeServersReqd == len(activeServers) {
			fmt.Println("No membership changes required")
			continue
		}

		// Trigger config change if lengths do not match
		if activeServersReqd > len(activeServers) {
			numServersToAdd := activeServersReqd - len(activeServers)
			for i := 0; i < len(serverList); i++ {
				if numServersToAdd == 0 {
					break
				}
				fmt.Println("checking server ", i)
				if !isElementPresentInArray(activeServers, i) && !isElementPresentInArray(unreachableServers, i) {
					fmt.Println("adding server", i, "to list of active servers")
					activeServers = append(activeServers, i)
					numServersToAdd--
				}
			}
		} else if activeServersReqd < len(activeServers) {
			activeServers = activeServers[:activeServersReqd]
		}

		if activeServersReqd != len(activeServers) {
			fmt.Println("Servers required = ", activeServersReqd, "Actual active servers = ", len(activeServers))
		}

		fmt.Println("Membership changes ------ new active and unreachable servers are ", activeServers, unreachableServers)
		// call config change on server
		if kv739_changeMembership(activeServers) == -1 {
			fmt.Println("Change membership is not successful to the given configuration ", activeServers)
		} else {
			fmt.Println("Change membership is successful to the given configuration", activeServers)
		}
	}
}
