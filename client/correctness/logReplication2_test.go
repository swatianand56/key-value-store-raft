package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Log replication — n servers failed (rest servers should get log replication)
func TestLogReplication2(t *testing.T) {
	var path = "./../../server/"
	var activeServerFilename = path + "activeServers.cfg"

	configStr := "0,1,2,3,4"
	serversToKill := "0,1"
	restServers := "2,3,4"

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
	arr := strings.Split(configStr, ",")                  // 2n+1 servers
	serversToKillArr := strings.Split(serversToKill, ",") // kill 2 servers
	restServersArr := strings.Split(restServers, ",")     // 3 servers

	majority := int(math.Ceil(float64(len(arr)+1) / 2))

	ServerSetup(0, serverList, arr)

	time.Sleep(time.Duration(5) * time.Second) // time to elect the leader

	var number_of_keys = 50

	kv739_init(serverList, 11)
	var oldValue string

	var keys_not_found = 0
	var key_value_mismatch = 0
	var put_key_unsuccessful = 0

	for i := 0; i < number_of_keys; i++ {
		var key = strconv.Itoa(i)
		x := kv739_put(key, key, &oldValue)
		y := kv739_get(key, &oldValue)
		if x != -1 {
			if y == -1 {
				keys_not_found++
			}
			if key != oldValue {
				key_value_mismatch++
			}
		} else {
			put_key_unsuccessful++
		}
	}

	time.Sleep(time.Duration(1) * time.Second) // buffer time to sync logs int all the servers

	ServerTeardown(serversToKillArr) // kill n servers

	time.Sleep(time.Duration(3) * time.Second) // buffer time to elect new leader

	for i := 0; i < number_of_keys; i++ {
		var key = strconv.Itoa(i)
		x := kv739_put(key, key, &oldValue)
		y := kv739_get(key, &oldValue)
		if x != -1 {
			if y == -1 {
				keys_not_found++
			}
			if key != oldValue {
				key_value_mismatch++
			}
		} else {
			put_key_unsuccessful++
		}
	}

	currentLeaderIndex := strconv.Itoa(leaderIndex)

	time.Sleep(time.Duration(1) * time.Second) // buffer time to sync logs int all the servers

	var content = make(map[string][]string)
	leaderLogSize := 0
	for i := range arr {
		index := arr[i]
		filePath := "./log-" + index + ".txt"
		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			t.Errorf("error in reading log file")
			ServerTeardown(restServersArr)
			return
		}
		lines := strings.Split(string(fileContent), "\n")
		content[index] = lines
		if index == currentLeaderIndex {
			leaderLogSize = len(lines)
		}
	}

	count := 0
	for i := range arr {
		index := arr[i]
		if len(content[index]) == leaderLogSize {
			count++
		} else {
			fmt.Println("length of logs", index, len(content[index]), leaderLogSize)
		}
	}
	if count < majority {
		fmt.Println("Failed test case: majority of servers should have the same length of logs as leader", count)
		t.Errorf("Failed test case: majority of servers should have the same length of logs as leader")
		ServerTeardown(restServersArr)
		return
	}

	for i := 0; i < leaderLogSize; i++ {
		count := 1
		line := content[strconv.Itoa(leaderIndex)][i]
		for j := range arr {
			index := arr[j]
			if index != currentLeaderIndex {
				if i < len(content[index]) && line == content[index][i] {
					count++
				}
			}
		}
		if count < majority {
			t.Errorf("Failed test case: majority of servers log content is not matching")
			ServerTeardown(restServersArr)
			return
		}
	}

	if keys_not_found > 0 || key_value_mismatch > 0 {
		t.Errorf("Failed test case: Keys not found => %d, keys value mismatch => %d", keys_not_found, key_value_mismatch)
	}

	if put_key_unsuccessful > 0 {
		fmt.Println("Unsuccessful Put Keys => ", put_key_unsuccessful)
	}

	ServerTeardown(restServersArr)
}
