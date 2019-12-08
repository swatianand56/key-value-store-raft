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

// Leader Election: Leader elected, kill, elect, kill (11 replicas — 5 times possible) - sleep for max electiontimeout (ping all servers — everyone should have same leader)
func TestLeaderElection3(t *testing.T) {
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

	kv739_init(serverList, 11)
	var number_of_keys = 10
	var oldValue string

	var keys_not_found = 0
	var key_value_mismatch = 0
	var put_key_unsuccessful = 0
	for i := 0; i < number_of_keys; i++ {
		var key = strconv.Itoa(i)
		x := kv739_put(key, key, &oldValue)
		if x != -1 {
			leaderServer := []string{strconv.Itoa(leaderIndex)}
			ServerTeardown(leaderServer)               // kill the leader
			time.Sleep(time.Duration(1) * time.Second) // buffer time to elect new leader

			y := kv739_get(key, &oldValue)
			if y == -1 {
				keys_not_found++
			}
			if key != oldValue {
				key_value_mismatch++
			}
			ServerSetup(0, serverList, leaderServer)          // up the server as follower
			time.Sleep(time.Duration(300) * time.Millisecond) // buffer time to up the server
		} else {
			put_key_unsuccessful++
		}
	}

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
