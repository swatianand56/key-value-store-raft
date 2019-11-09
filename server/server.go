package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
)

var numServers = 3

var majoritySize = math.Ceil(float64(numServers+1) / 2)

// Make this dynamic initialization based on numServers
var config = []map[string]string{
	{
		"port":     "8001",
		"host":     "localhost",
		"filename": "./0.txt",
		"logfile":  "./log-0.txt",
	},
	{
		"port":     "8002",
		"host":     "localhost",
		"filename": "./1.txt",
		"logfile":  "./log-1.txt",
	},
	{
		"port":     "8003",
		"host":     "localhost",
		"filename": "./2.txt",
		"logfile":  "./log-2.txt",
	},
}

var serverIndex int

var leaderIndex int

// Convert to epoch timestamp in ms
var lastHeartbeatFromLeader int

// Time in ms
// May be check the timeout size effect on the overall system performance
var timeoutMin = 50
var timeoutMax = 200
var leaderElectionTimeout int
var leaderHeartBeatDuration = 10

// This entry goes in the log file
type LogEntry struct {
	Key, Value, termId, indexId string
}

// This entry goes in the state machine on commit
type KeyValuePair struct {
	Key, Value string
}

type Task int

// ... Leader only servers this request always
// If this server not the leader, redirect the request to the leader (How can client library know who the leader is?)
func (t *Task) GetKey(key string, value *string) error {
	// read the value of the key from the file and return
	return nil
}

//PutKey ...
// The request should only come to the leader. If this is not the leader redirect the request to the leader
func (t *Task) PutKey(keyValue KeyValuePair, oldValue *string) error {
	// Make entry in the log file
	// Call the LogReplication function and take care of steps 2,3,4 there
	// 2. Send AppendEntries to all other servers in async
	// 3. When majority response received, add to the server file
	// 4. If AppendEntries fail, need to update the lastMatchedLogIndex for that server
	// Update the last commit ID? -- Do we need it as a variable?
	// return response to the client
	return nil
}

// AppendEntries ...
// Heartbeat to inform followers to commit
// Heartbeat to sync followers
// Should appendEntries be able to declare that it's now the leader? -- like the caller passes its own serverId? Its only called by the leader anyway
// Update lastheardfromleader time
// If possible update the timeout time for leader election
// ...
func (t *Task) AppendEntries(lastTermId int, lastIndexId int, commitId int, keyvaluepair KeyValuePair) error {
	return nil
}

func (t *Task) RequestVote(serverIndex int, serverTerm int, lastLogIndex int) error {
	return nil
}

// candidate election, request vote
func LeaderElection() error {
	// make this func async
	// in an infinite loop, check if currentTime - lastHeartbeatFromLeader > leaderElectionTimeout, initiate election
	// sleep for leaderElectionTimeout
	// probably need to extend the sleep/waiting time everytime lastHeartbeatfromleader is received (variable value can change in AppendEntries)

	// For leaderElection: Make RequestVote RPC calls to all other servers. (count its own vote +1) -- Do this async
	// If majority true response received, change leaderIndex = serverIndex
	return nil
}

// Call from LeaderElection?
func CheckConsistencySafety() error {
	return nil
}

// Called from PutKey, Also during heartbeat from leader every leaderHeartBeatDuration
// 2. Send AppendEntries to all other servers in async
// 3. When majority response received, add to the server file
// 4. If AppendEntries fail, need to update the lastMatchedLogIndex for that server
func LogReplication() error {
	return nil
}

//Init ... takes in config and index of the current server in config
func Init(index int) error {
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

	// TODO: call leader election function asynchronously so that it's non blocking
	return nil
}

func main() {
	args := os.Args[1:]
	serverIndex, _ = strconv.Atoi(args[0])
	// TODO: initialize the leaderElectionTimeout time randomly selected between timeoutmin and timeoutmax
	pid := os.Getpid()
	fmt.Printf("Server %d starts with process id: %d\n", serverIndex, pid)
	filename := config[serverIndex]["filename"]
	_, err := os.Stat(filename)

	// TODO: Create server file and log file if not already present
	if os.IsNotExist(err) {
		_, err := os.OpenFile(filename, os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create file ", err)
		}
	}

	err = Init(serverIndex)
}
