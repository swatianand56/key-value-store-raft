package main

import (
	"fmt"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

func isElementPresentInArray(arr []int, ele int) bool {
	for _, i := range arr {
		if i == ele {
			return true
		}
	}
	return false
}

func main() {
	serverList := []string{
		"10.10.1.1:8001",
		"10.10.1.2:8002",
		"10.10.1.3:8003",
		"10.10.1.1:8004",
		"10.10.1.2:8005",
		"10.10.1.3:8006",
		"10.10.1.1:8007",
		"10.10.1.2:8008",
		"10.10.1.3:8009",
		"10.10.1.1:8010",
		"10.10.1.2:8011",
	}
	activeServers := []int{0, 1, 2}
	unreachableServers := []int{}
	failureTestTime := 10 // time in seconds
	failureHandleCapacity := 1
	activeServersReqd := 3
	const FailureCycles = 3
	numFailureCycles := FailureCycles
	serversToStart := []int{}
	serversToKill := []int{}

	for {
		fmt.Println("sleeping for 10 seconds")
		time.Sleep(time.Duration(failureTestTime) * time.Second)
		// start with 3 machines: least number, can handle the failure of 1 machine (if a machine goes down -- up another machine)
		// if n machines have failed in the last m minutes, increase failure tolerance by n+1 (max machines in our case = 11 -- this can tolerate the failure of 5 machines)
		// if 0 machines have failed in the last m minutes, decrease failure tolerance by 1 (min number of machines being 3)
		// ping all the active servers

		fmt.Println("active servers are ---- ", activeServers, failureHandleCapacity, unreachableServers)

		serversToStart = []int{}
		serversToKill = []int{}

		var serverStatus int
		numFailures := 0

		// make a check over unreachable servers and remove them from the list if they are now reachable

		// for index, server := range unreachableServers {
		// 	fmt.Println("checking unreachable server ", server)
		// 	if kv739_init([]string{serverList[server]}, 1) == 0 {
		// 		unreachableServers[index] = -1
		// 	}
		// }

		// sort.Slice(unreachableServers, func(i, j int) bool {
		// 	return unreachableServers[i] < unreachableServers[j]
		// })

		// for index, server := range unreachableServers {
		// 	if server == -1 {
		// 		unreachableServers = unreachableServers[:index]
		// 		break
		// 	}
		// }

		for index, server := range activeServers {
			fmt.Println("checking active server ", server)
			serverStatus = kv739_init([]string{serverList[server]}, 1)
			if serverStatus == -1 { // error connecting
				// unreachableServers = append(unreachableServers, server)
				activeServers[index] = -1
				numFailures++
				fmt.Println("failure encountered new active servers are ", activeServers, unreachableServers)
			}
		}

		sort.Sort(sort.Reverse(sort.IntSlice(activeServers)))

		for index, server := range activeServers {
			if server == -1 {
				activeServers = activeServers[:index]
				break
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
					serversToStart = append(serversToStart, i)
					numServersToAdd--
				}
			}
		} else if activeServersReqd < len(activeServers) {
			for i := activeServersReqd; i < len(activeServers); i++ {
				serversToKill = append(serversToKill, i)
			}
			activeServers = activeServers[:activeServersReqd]
		}

		fmt.Println("servers to start are ", serversToStart)

		if len(serversToStart) > 0 {
			for _, server := range serversToStart {
				cmd := exec.Command("ssh", strings.Split(serverList[server], ":")[0])
				err := cmd.Run()
				if err != nil {
					fmt.Println("could not ssh into another machine")
					continue
				}

				cmd = exec.Command("./key-value-store-raft/server/start-server.sh", strconv.Itoa(server))
				err = cmd.Run()
				if err != nil {
					fmt.Println("Error starting the server", err)
				}
				fmt.Println("server started", server)
			}
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
			if len(serversToKill) > 0 {
				for _, server := range serversToKill {
					cmd := exec.Command("ssh", strings.Split(serverList[server], ":")[0])
					err := cmd.Run()
					if err != nil {
						fmt.Println("could not ssh into another machine")
						continue
					}
					cmd = exec.Command("./key-value-store-raft/server/kill-server.sh", strconv.Itoa(server))
					err = cmd.Run()
					if err != nil {
						fmt.Println("Error starting the server", err)
					}
					fmt.Println("server started", server)
				}
			}
		}
	}
}
