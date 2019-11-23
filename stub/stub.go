package client

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
	leaderIndex = 0
	serverList = serverListArg
	address := serverList[leaderIndex]
	client, err = rpc.DialHTTP("tcp", address)
	if err != nil {
		fmt.Println("Connection error: ", err)
		return -1
	}
	return 0
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

//export kv739_get
func kv739_get(key string, value *string) int {
	err := client.Call("Task.GetKey", key, value)
	if err != nil {
		if match, _ := regexp.MatchString(".*LeaderIndex.*", err.Error()); match {
			//Retry logic
			leaderIndex, _ = strconv.Atoi(strings.Split(err.Error(), "LeaderIndex:")[1])
			address := serverList[leaderIndex]
			client, err = rpc.DialHTTP("tcp", address)
			if err == nil {
				err = client.Call("Task.GetKey", key, value)
				if err == nil {
					if len(*value) > 0 {
						return 0
					}
					return 1
				}
				fmt.Println("Unable to get key: ", key, " from leader: ", leaderIndex, " err: ", err)
			}
			fmt.Println("Unable to establish connection with leader: ", leaderIndex, err)
		} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
			for index, server := range serverList {
				if index != leaderIndex {
					fmt.Println("Retrying to establish connection with server index ", index)
					client, err = rpc.DialHTTP("tcp", server)
					if err == nil {
						err = client.Call("Task.GetKey", key, value)
						if err == nil {
							leaderIndex = index
							if len(*value) > 0 {
								return 0
							}
							return 1
						}
						fmt.Println("Unable to get key: ", key, " from server: ", server, " err: ", err)
					}
					fmt.Println("Unable to establish connection with server: ", server, err)
				}
			}
		}
		fmt.Println("Could not get key: ", key, " value: ", value, "err: ", err)
		return -1
	}
	if len(*value) > 0 {
		return 0
	}
	return 1
}

//export kv739_put
func kv739_put(key string, value string, oldValue *string) int {
	err := client.Call("Task.PutKey", KeyValuePair{Key: key, Value: value}, oldValue)
	if err != nil {
		if match, _ := regexp.MatchString(".*LeaderIndex.*", err.Error()); match {
			//Retry logic
			leaderIndex, _ = strconv.Atoi(strings.Split(err.Error(), "LeaderIndex:")[1])
			address := serverList[leaderIndex]
			client, err = rpc.DialHTTP("tcp", address)
			if err == nil {
				err = client.Call("Task.PutKey", KeyValuePair{Key: key, Value: value}, oldValue)
				if err == nil {
					if len(*oldValue) > 0 {
						return 0
					}
					return 1
				}
				fmt.Println("Unable to get key: ", key, " from leader: ", leaderIndex, " err: ", err)
			}
			fmt.Println("Unable to establish connection with leader: ", leaderIndex, err)
		} else if match, _ := regexp.MatchString(".*connection.*", err.Error()); match {
			if len(serverList) > 1 {
				for index, server := range serverList {
					if index != leaderIndex {
						fmt.Println("Retrying to establish connection with server index ", index)
						client, err = rpc.DialHTTP("tcp", server)
						if err == nil {
							err := client.Call("Counter.PutKey", KeyValuePair{Key: key, Value: value}, &oldValue)
							if err == nil {
								leaderIndex = index
								if len(*oldValue) > 0 {
									return 0
								}
								return 1
							}
							fmt.Println("Unable to put key: ", key, " on server: ", server, " err: ", err)
						} else {
							fmt.Println("Unable to establish connection with server: ", server, err)
						}
					}
				}
			}
		} else {
			fmt.Println("Could not put key: ", key, " value: ", value, " err: ", err)
		}

		return -1
	}

	if len(*oldValue) > 0 {
		return 0
	}
	return 1
}

func main() {
}
