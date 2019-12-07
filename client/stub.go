package main

import (
	"fmt"
	"net"
	"net/rpc"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var err error
var client *rpc.Client
var leaderIndex int
var reply string
var conn net.Conn
var serverList []string

//export kv739_init
func kv739_init(serverListArg []string, length int) int {
	//TODO: can you work without length argument?
	serverList = serverListArg
	leaderIndex = 0
	return 0
}

//export kv739_shutdown
func kv739_shutdown() int {
	return 0
}

func kv739_changeMembership(newConfig []int) int {
	return executeChangeMembership(newConfig, serverList[leaderIndex])
}

func executeChangeMembership(newConfig []int, address string) int {
	if len(address) > 0 {
		conn, err = net.DialTimeout("tcp", address, 250*time.Millisecond)
	}
	if err == nil {
		client = rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(250 * time.Millisecond))
		var reply int
		err = client.Call("RaftServer.ChangeMembership", newConfig, &reply)
		if err == nil {
			return 0
		} else if match, _ := regexp.MatchString(".*LeaderIndex.*", err.Error()); match {
			thisLeader, _ := strconv.Atoi(strings.Split(err.Error(), "LeaderIndex:")[1])
			if thisLeader != -1 {
				leaderIndex = thisLeader
			}
			address = serverList[leaderIndex]
			return executeChangeMembership(newConfig, address)
		} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
			leaderIndex = (leaderIndex + 1) % len(serverList)
			getResult := executeChangeMembership(newConfig, serverList[leaderIndex])
			if getResult != -1 {
				return getResult
			}
		}
		fmt.Println("Unable to change membership: ", newConfig, " from leader: ", leaderIndex, " err: ", err)
	} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
		leaderIndex = (leaderIndex + 1) % len(serverList)
		getResult := executeChangeMembership(newConfig, serverList[leaderIndex])
		if getResult != -1 {
			return getResult
		}
	}
	fmt.Println("Error in change membership", err)
	return -1
}

func executeGetKey(key string, value *string, address string) int {
	if len(address) > 0 {
		conn, err = net.DialTimeout("tcp", address, 250*time.Millisecond)
	}
	if err == nil {
		client = rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(250 * time.Millisecond))
		err = client.Call("RaftServer.GetKey", key, value)
		if err == nil {
			if len(*value) > 0 {
				return 0
			}
			return 1
		} else if match, _ := regexp.MatchString(".*LeaderIndex.*", err.Error()); match {
			thisLeader, _ := strconv.Atoi(strings.Split(err.Error(), "LeaderIndex:")[1])
			if thisLeader != -1 {
				leaderIndex = thisLeader
			}
			address = serverList[leaderIndex]
			return executeGetKey(key, value, address)
		} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
			leaderIndex = (leaderIndex + 1) % len(serverList)
			getResult := executeGetKey(key, value, serverList[leaderIndex])
			if getResult != -1 {
				return getResult
			}
		}
		fmt.Println("Unable to get key: ", key, " from leader: ", leaderIndex, " err: ", err)
	} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
		leaderIndex = (leaderIndex + 1) % len(serverList)
		getResult := executeGetKey(key, value, serverList[leaderIndex])
		if getResult != -1 {
			return getResult
		}
	}
	fmt.Println("Error in get key", err)
	return -1
}

//export kv739_get
func kv739_get(key string, value *string) int {
	getResult := executeGetKey(key, value, serverList[leaderIndex])
	if getResult == -1 {
		fmt.Println("Could not get key: ", key, " value: ", value, "err: ", err)
		return -1
	}
	if len(*value) > 0 {
		return 0
	}
	return 1
}

func executePutKey(key string, value string, oldValue *string, address string) int {
	fmt.Println("calling " + key)
	if len(address) > 0 {
		conn, err = net.DialTimeout("tcp", address, 250*time.Millisecond)
	}

	if err == nil {
		client = rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(250 * time.Millisecond))
		err = client.Call("RaftServer.PutKey", KeyValuePair{Key: key, Value: value}, oldValue)
		if err == nil {
			if len(*oldValue) > 0 {
				return 0
			}
			return 1
		} else if match, _ := regexp.MatchString(".*LeaderIndex.*", err.Error()); match {
			thisLeader, _ := strconv.Atoi(strings.Split(err.Error(), "LeaderIndex:")[1])
			if thisLeader != -1 {
				leaderIndex = thisLeader
			}
			address = serverList[leaderIndex]
			return executePutKey(key, value, oldValue, address)
		} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
			// TODO doesn't the server reply with the new leader?  Shouldn't we read the response from the reply instead of iterating through the server list?
			leaderIndex = (leaderIndex + 1) % len(serverList)
			putResult := executePutKey(key, value, oldValue, serverList[leaderIndex])
			if putResult != -1 {
				return putResult
			}
		}
		fmt.Println("Unable to put key: ", key, " from leader: ", leaderIndex, " err: ", err)
	} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
		leaderIndex = (leaderIndex + 1) % len(serverList)
		putResult := executePutKey(key, value, oldValue, serverList[leaderIndex])
		if putResult != -1 {
			return putResult
		}
	}
	fmt.Println("Unable to establish connection with leader: ", leaderIndex, err)
	return -1
}

//export kv739_put
func kv739_put(key string, value string, oldValue *string) int {
	getResult := executePutKey(key, value, oldValue, serverList[leaderIndex])
	return getResult
}
