package main

import (
	"fmt"
	"net/rpc"
	"regexp"
	"strconv"
	"strings"
)

var err error
var client *rpc.Client
var leaderIndex int
var reply string
var serverList []string

//KeyValuePair ... interface type
type KeyValuePair struct {
	Key, Value string
}

//export kv739_init
func kv739_init(serverListArg []string, length int) int {
	//TODO: can you work without length argument?
	for index := range serverListArg {
		serverList = serverListArg
		address := serverList[index]
		leaderIndex = index
		client, err = rpc.DialHTTP("tcp", address)
		if err != nil {
			if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
				fmt.Println("Connection error: ", err)
				continue
			} else {
				return -1
			}
		}
		return 0
	}
	return -1
}

//export kv739_shutdown
func kv739_shutdown() int {
	err := client.Close()
	if err != nil {
		fmt.Println("Unable to shutdown client connection: ", err)
		return -1
	}
	return 0
}

func executeGetKey(key string, value *string, address string) int {
	if len(address) > 0 {
		client, err = rpc.DialHTTP("tcp", address)
	}
	if err == nil {
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
			for index, server := range serverList {
				if index != leaderIndex {
					getResult := executeGetKey(key, value, server)
					if getResult != -1 {
						return getResult
					}
				}
			}
		}
		fmt.Println("Unable to get key: ", key, " from leader: ", leaderIndex, " err: ", err)
	} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
		for index, server := range serverList {
			if index != leaderIndex {
				getResult := executeGetKey(key, value, server)
				if getResult != -1 {
					return getResult
				}
			}
		}
	}
	return -1
}

//export kv739_get
func kv739_get(key string, value *string) int {
	getResult := executeGetKey(key, value, "")
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
	if len(address) > 0 {
		client, err = rpc.DialHTTP("tcp", address)
	}

	if err == nil {
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
			for index, server := range serverList {
				if index != leaderIndex {
					putResult := executePutKey(key, value, oldValue, server)
					if putResult != -1 {
						return putResult
					}
				}
			}
		}
		fmt.Println("Unable to put key: ", key, " from leader: ", leaderIndex, " err: ", err)
	} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
		for index, server := range serverList {
			if index != leaderIndex {
				putResult := executePutKey(key, value, oldValue, server)
				if putResult != -1 {
					return putResult
				}
			}
		}
	}
	fmt.Println("Unable to establish connection with leader: ", leaderIndex, err)
	return -1
}

//export kv739_put
func kv739_put(key string, value string, oldValue *string) int {
	getResult := executePutKey(key, value, oldValue, "")
	if getResult == -1 {
		fmt.Println("Could not put key: ", key, " value: ", value, "err: ", err)
		return -1
	}

	if len(*oldValue) > 0 {
		return 0
	}
	return 1
}
