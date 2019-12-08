package main

import (
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Log replication — n servers failed (rest servers should get log replication)
func TestLogReplication3(t *testing.T) {
	var path = "./../../server/"
	var activeServerFilename = path + "activeServers.cfg"

	configStr := "0,1,2,3,4"
	serversToKill := "0,1,2"
	restServers := "3,4"

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
	totalServers := strings.Split(configStr, ",")         // 2n+1 servers
	serversToKillArr := strings.Split(serversToKill, ",") // kill 2 servers
	restServersArr := strings.Split(restServers, ",")     // 3 servers

	ServerSetup(0, serverList, totalServers)

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

	failed_put_keys_req := 0
	failed_get_keys_req := 0
	number_of_keys = 5
	for i := 0; i < number_of_keys; i++ {
		var key = strconv.Itoa(i)
		x := kv739_put(key, key, &oldValue)
		y := kv739_get(key, &oldValue)
		if y == -1 {
			failed_get_keys_req++
		}
		if x == -1 {
			failed_put_keys_req++
		}
	}

	if failed_get_keys_req != number_of_keys || failed_put_keys_req != number_of_keys {
		t.Error("Put key and get key should fail")
	}
	ServerTeardown(restServersArr) // kill n+1 servers
}
