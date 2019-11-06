package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"time"
)

// We would want to change the number of servers (while testing)
var config = []map[string]string{
	{
		"port":     "8001",
		"host":     "localhost",
		"filename": "./0.txt",
		"logfile": "./log-0.txt",
	},
	{
		"port":     "8002",
		"host":     "localhost",
		"filename": "./1.txt",
		"logfile": "./log-1.txt",
	},
	{
		"port":     "8003",
		"host":     "localhost",
		"filename": "./2.txt",
		"logfile": "./log-2.txt",
	},
}
var serverIndex int

var leaderIndex int

var lastHeartbeatFromLeader int

// This entry goes in the log file
type LogEntry struct {
	Key, Value, termId, indexId string
}

// This entry goes in the state machine on commit
type KeyValuePair struct {
	Key, Value string
}

type Task int

func (t *Task) GetKey(key string, value *string) error {
	return nil
}

//PutKey ...
func (t *Task) PutKey(keyValue KeyValuePair, oldValue *string) error {
	return nil
}

// AppendEntries ... 
// Heartbeat to inform followers to commit
// Heartbeat to sync followers
// ...
func (t *Task) AppendEntries(lastTermId int, lastIndexId int, commitId int, keyvaluepair KeyValuePair) error {
	return nil
}

func (t *Task) RequestVote(serverIndex int, serverTerm int, lastLogIndex int) error {
	return nil
}

// candidate election, request vote
func LeaderElection() error {
	return nil
}

// Call from LeaderElection?
func CheckConsistencySafety() error {
	return nil
}

// Called from PutKey
func LogReplication() error {
	return nil
}

//Init ... takes in config and index of the current server in config
func Init(index int) error {
	// create file and add first line if not already present

	t := new(Task)

	err := rpc.Register(&t)

	if err != nil {
		fmt.Println("Format of service Task isn't correct. ", err)
	}

	// Register a HTTP handler
	rpc.HandleHTTP()

	// Listen to TPC connections on port 1234
	listener, e := net.Listen("tcp", config[index]["host"]+":"+config[index]["port"])
	if e != nil {
		fmt.Println("Listen error: ", e)
	}
	log.Printf("Serving RPC server on port %d", config[index]["port"])

	// Start accept incoming HTTP connections
	err = http.Serve(listener, nil)
	if err != nil {
		fmt.Println("Error serving: ", err)
	}
	return nil
}

// initialize timeout
// while (true), run leader election when timeout

func main() {
	args := os.Args[1:]
	serverIndex, _ = strconv.Atoi(args[0])
	pid := os.Getpid()
	fmt.Printf("Server %d starts with process id: %d\n", serverIndex, pid)
	filename := config[serverIndex]["filename"]
	_, err := os.Stat(filename)

	if os.IsNotExist(err) {
		_, err := os.OpenFile(filename, os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create file ", err)
		}
		curTimeStamp := strconv.FormatInt(time.Now().UnixNano(), 10)
		err = ioutil.WriteFile(filename, []byte(curTimeStamp), 0)
	}

	err = Init(serverIndex)
}
