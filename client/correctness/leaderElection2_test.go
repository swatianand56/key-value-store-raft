package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/rpc"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Leader election success - sleep for max electiontimeout (ping all servers — everyone should have same leader)
func TestLeaderElection2(t *testing.T) {
	var path = "./../../server/"
	var activeServerFilename = path + "activeServers.cfg"

	configStr := "0,1,2,3,4"

	serverList := []string{
		"localhost:8001",
		"localhost:8002",
		"localhost:8003",
		"localhost:8004",
		"localhost:8005",
	}

	err := ioutil.WriteFile(activeServerFilename, []byte(configStr), 0)
	if err != nil {
		t.Errorf("Failed to write in server config file %s", err)
		return
	}
	arr := strings.Split(configStr, ",") // 2n+1 servers
	majority := int(math.Ceil(float64(len(arr)+1) / 2))

	ServerSetup(0, serverList, arr)

	time.Sleep(time.Duration(5) * time.Second) // time to elect the leader

	var leaderIndexes = make([]int, 0)
	for i := range arr {
		index, _ := strconv.Atoi(arr[i])
		// CurrentState
		conn, err = net.DialTimeout("tcp", serverList[index], 250*time.Millisecond)
		if err == nil {
			client = rpc.NewClient(conn)
			defer client.Close()
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(250 * time.Millisecond))
			var oldValue string
			err = client.Call("RaftServer.GetKey", "a", &oldValue)
			if err == nil {
				leaderIndexes = append(leaderIndexes, index)
				fmt.Println("leader is => ", index)
			} else if match, _ := regexp.MatchString(".*LeaderIndex.*", err.Error()); match {
				thisLeader, _ := strconv.Atoi(strings.Split(err.Error(), "LeaderIndex:")[1])
				leaderIndexes = append(leaderIndexes, thisLeader)
				fmt.Println("leader is => ", thisLeader)
			}
		} else {
			t.Errorf("Fail test case: failed to make tcp connection with server %d", index)
		}
	}

	sort.Slice(leaderIndexes, func(i, j int) bool {
		return leaderIndexes[i] < leaderIndexes[j]
	})

	j := 0
	for i := 0; i < len(leaderIndexes); i++ {
		if leaderIndexes[i] >= 0 {
			j = i
			break
		}
	}
	found := false
	count := 1
	if j < len(leaderIndexes) {
		leaderIndex := leaderIndexes[j]
		for i := j + 1; i < len(leaderIndexes); i++ {
			if leaderIndexes[i] == leaderIndex {
				count++
			} else {
				if count < majority {
					count = 1
					leaderIndex = leaderIndexes[i]
				} else {
					found = true
					break
				}
			}
		}
	}

	if !found && count < majority {
		t.Errorf("Fail test case: Majority of cluster does not have same leader")
	}
	ServerTeardown(arr)
}
