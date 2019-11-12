// see the key-value-distributed to see how rpcs are implemented.
// nick: Leader election
// nick: request vote
// append entries
// consistency checks
// client input
// unit tests

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

// Persistent state on all servers: currentTerm, votedFor (candidate ID that received vote in current term)-- not sure if we need this
//var serverVotedFor int --- These 2 have to be added in logFile (may be use the first 2 lines of the logFile)
//var currentTerm int

const numServers = 3

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

// index of the last committed entry
var commitIndex int

var lastAppliedIndex int

// Convert to epoch timestamp in ms
var lastHeartbeatFromLeader int

// this is to define which values to send for replication to each server during AppendEntries (initialize to last log index on leader + 1)
var nextIndex [numServers]int // would be init in leader election

// not sure where this is used -- but keeping it for now (initialize to 0)
var matchIndex [numServers]int // would be init in leader election

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
	// do log replication and commit on get entries to get the most consistent response in case of leader failure
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
	// Keep trying indefinitely unless the entry is replicated on all servers
	// Update the last commit ID? -- Do we need it as a variable?, replicate the data in log and lastAppliedIndex
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
// if leaderid != leaderID, change the leaderID index maintained on this server
// entries argument empty for heartbeat
// TODO: @Swati
func (t *Task) AppendEntries(leaderTerm int, leaderID int, prevLogIndex int, prevLogTerm int, entries []KeyValuePair, leaderCommitIndex int, currentTerm *int, success *bool) error {
	// if leaderCommitIndex > commitIndex on server, apply the commits to the log file (should only be done after replication and logMatching is successful)
	// if currentTerm > leaderTerm, return false
	// if term at server log prevLogIndex != prevLogTerm, return false
	// if matched, then delete all entries after prevLogIndex, and add the new entries.
	// Update commitIndex = min(last index in log, leaderCommitIndex)
	return nil
}

// need to maintain what was the last term voted for
// candidate term is the term proposed by candidate (current term + 1)
// if receiver's term is greater then, currentTerm is returned and candidate has to update it's term (to be taken care in leader election)

/*

These data points are required by the server and requestvote:

* Server:

- selfId: int
- currentTerm: int
- lastMessageTime: time (ms)
- electionMinTime: time (ms)
- electionMaxTime: time (ms)

- cluster: server[]
- server: (id, ip, port)

- log: term[]
- term: (termId, entry[])
- termId: int
- entry: (entryId, data)
- entryId: int

* RequestVote RPC:

"Plz vote for me as new leader!"

Sends async RPC to all clients.

- term := currentTerm
- candidateId := selfId
- lastLogTerm := log[-1].termId
- lastLogIndex := log[-1][-1].entryId

Receives election result:

- term: new currentTerm, if bigger.
- voteGranted?: bool

*/

func (t *Task) RequestVote(candidateIndex int, candidateTerm int, lastLogIndex int, lastLogTerm int, currentTerm *int, voteGranted *bool) error {
	// Reply false if candidateTerm < currentTerm : TODO: should this be < or <=
	// If votedFor is null or candidateIndex, and candidate's log is as up-to-date (greater term index, same term greater log index) then grant vote
	// also update the currentTerm to the newly voted term and update the votedFor
	return nil
}

func sendLeaderHeartbeats() error {
	// if current server is the leader, then send AppendEntries heartbeat RPC at idle times to prevent leader election and just after election
	return nil
}

func applyCommittedEntries() error {
	// at an interval of t ms, if (commitIndex > lastAppliedIndex), then apply entries one by one to the server file (state machine)
	return nil
}

// candidate election, request vote
func LeaderElection() error {
	// make this func async
	// in an infinite loop, check if currentTime - lastHeartbeatFromLeader > leaderElectionTimeout, initiate election
	// sleep for leaderElectionTimeout
	// probably need to extend the sleep/waiting time everytime lastHeartbeatfromleader is received (variable value can change in AppendEntries)

	// For leaderElection: Increment currentTerm
	// Make RequestVote RPC calls to all other servers. (count its own vote +1) -- Do this async
	// If majority true response received, change leaderIndex = serverIndex
	// Also reinit entries for matchIndex = 0 and nextIndex = last log index + 1 array for all servers
	// Send AppendEntires Heartbeat RPC to all servers to announce the leadership, also call sendLeaderHeartbeats() asynchronously from here and just let it running
	// if receiver's term is greater then, currentTerm is returned, update your current term and continue
	return nil
}

// Call from LeaderElection? -- Not sure if we are using this
func CheckConsistencySafety() error {
	return nil
}

// Called from PutKey, Also during heartbeat from leader every leaderHeartBeatDuration
// 2. Send AppendEntries to all other servers in async
// 3. When majority response received, add to the server file
// 4. If AppendEntries fail, need to update the lastMatchedLogIndex for that server and retry. this keeps retrying unless successful
// TODO: @Geetika
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
	// call applyCommittedEntries asynchronously so that it keeps running in the background
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
