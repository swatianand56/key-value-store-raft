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
	serverList = serverListArg
	leaderIndex = 0
	if len(serverList) == 0 {
		return -1
	}
	address := serverList[0]
	conn, err = net.DialTimeout("tcp", address, 250*time.Millisecond)
	if err != nil {
		fmt.Println("Connection error: ", err)
		return -1
	}
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
		conn, err = net.DialTimeout("tcp", address, 1000*time.Millisecond)
	}
	if err == nil {
		client = rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(1000 * time.Millisecond))
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
