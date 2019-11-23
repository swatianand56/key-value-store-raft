// see the key-value-distributed to see how rpcs are implemented.
// nick: Leader election
// nick: request vote
// append entries
// consistency checks
// client input
// unit tests

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
)

const numServers = 3

var majoritySize = int(math.Ceil(float64(numServers+1) / 2))

// Make this dynamic initialization based on numServers

var config = []map[string]string{
	{
		"port":     "8001",
		"host":     "localhost",
		"filename": "./0.txt",
		"logfile":  "./log-0.txt",
		"metadata": "./metadata-0.txt",
	},
	{
		"port":     "8002",
		"host":     "localhost",
		"filename": "./1.txt",
		"logfile":  "./log-1.txt",
		"metadata": "./metadata-1.txt",
	},
	{
		"port":     "8003",
		"host":     "localhost",
		"filename": "./2.txt",
		"logfile":  "./log-2.txt",
		"metadata": "./metadata-2.txt",
	},
}

// Persistent state on all servers: currentTerm, votedFor (candidate ID that received vote in current term)-- using a separate metadata file for this
// var serverVotedFor int --- These 2 have to be added in logFile (may be use the first 2 lines of the logFile)
// var currentTerm int (Also maintaining it in volatile memory to avoid Disk IO everytime)

var serverCurrentTerm int // start value could be 0 (init from persistent storage during start)

var serverVotedFor int // start value could be -1 (init from persistent storage during start)

var serverIndex int

var leaderIndex int

// DOUBT: index of the last committed entry (should these 2 be added to persistent storage as well? How are we going to tell till where is the entry applied in the log?)
var commitIndex int

var lastAppliedIndex int

// init from persistent storage on start
var lastLogEntryIndex int

// Convert to epoch timestamp in ms
var lastHeartbeatFromLeader int64

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

var me RaftServer

type RaftServer struct {
	// who am I?
	id    int
	alive bool

	// when is it?
	currentTerm     int
	lastMessageTime int64
	responses       []Vote

	// elections
	lastVotedTerm        int
	leader               int
	electionMinTime      int
	electionMaxTime      int
	discardElectionCycle bool

	// log
	cluster []Server
	log     []Term
}

type Term struct {
	log    []Entry
	termId int
}

type Entry struct {
	key, value string
}

type Server struct {
	id, port int
	ip       []byte
}

type Vote struct {
	source  int
	target  int
	granted bool
}

//LogEntry ... This entry goes in the log file
// get operations are denoted by using blank value
type LogEntry struct {
	Key, Value, TermID, IndexID string
}

//KeyValuePair ... This entry goes in the state machine on commit
type KeyValuePair struct {
	Key, Value string
}

type AppendEntriesArgs struct {
	leaderTerm, leaderID, prevLogIndex, prevLogTerm int
	entries                                         []LogEntry
	leaderCommitIndex                               int
}

type AppendEntriesReturn struct {
	currentTerm int
	success     bool
}

type RequestVoteArgs struct {
	candidateIndex int
	candidateTerm  int
	lastLogIndex   int
	lastLogTerm    int
}

type RequestVoteResponse struct {
	currentTerm int
	voteGranted bool
}

//Task ... type Task to be using in the context of RPC messages
type Task int

//GetKey ... Leader only servers this request always
// If this server not the leader, redirect the request to the leader (How can client library know who the leader is?)
func (t *Task) GetKey(key string, value *string) error {
	if serverIndex == leaderIndex {
		// read the value of the key from the file and return
		// do log replication and commit on get entries to get the most consistent response in case of leader failure

		// first add the entry to the log of the leader, then call logReplication, it returns when replicated to majority
		// once returned by logreplication, commit to file and return success, also update the commitIndex and lastappliedindex

		lastLogEntryIndex++                // what about 2 parallel requests trying to use the same logEntryIndex (this is a shared variable -- do we need locks here?)
		logEntryIndex := lastLogEntryIndex // these 2 steps should be atomic

		logentrystr := string("\n" + key + ",," + strconv.Itoa(serverCurrentTerm) + strconv.Itoa(logEntryIndex))
		err := writeEntryToLogFile(logentrystr)
		if err != nil {
			return err
		}

		err = LogReplication()
		if err != nil {
			fmt.Println("Unable to replicate the get request to a majority of servers", err)
			return err
		}
		// if successful, apply log entry to the file (fetch the value of the key from the file for get)
		commitIndex = int(math.Max(float64(commitIndex), float64(logEntryIndex)))
		lastAppliedIndex = commitIndex
		// fetch the value of the key

		file, err := os.Open(config[serverIndex]["filename"])
		if err != nil {
			fmt.Println(err)
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var line []string = strings.Split(scanner.Text(), ",")
			if line[0] == key {
				*value = line[1]
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("GetKey: Error while reading the file to find key value ", err)
			return err
		}
		return nil
	}
	// cannot find a way to transfer connection to another server, so throwing error with leaderIndex
	return errors.New("LeaderIndex:" + strconv.Itoa(leaderIndex))
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
	if serverIndex == leaderIndex {
		lastLogEntryIndex++                // what about 2 parallel requests trying to use the same logEntryIndex (this is a shared variable -- do we need locks here?)
		logEntryIndex := lastLogEntryIndex // these 2 steps should be atomic

		logentrystr := string("\n" + keyValue.Key + "," + keyValue.Value + "," + strconv.Itoa(serverCurrentTerm) + strconv.Itoa(logEntryIndex))
		err := writeEntryToLogFile(logentrystr)
		if err != nil {
			return err
		}
		// what if failure happens, do we decrement the lastLogEntryIndex? what if another request has updated it already -- there might be a conflict here
		err = LogReplication()
		if err != nil {
			fmt.Println("Unable to replicate the get request to a majority of servers", err)
			return err
		}

		// Written to log, now do the operation on the file (following write-ahead logging semantics)
		keyFound := false
		newKeyValueString := keyValue.Key + "," + keyValue.Value
		filename := config[serverIndex]["filename"]
		fileContent, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Println(err)
			return err
		}
		lines := strings.Split(string(fileContent), "\n")

		for i := 1; i < len(lines); i++ {
			line := strings.Split(lines[i], ",")
			if line[0] == keyValue.Key {
				*oldValue = line[1]
				lines[i] = newKeyValueString
				keyFound = true
				break
			}
		}

		if !keyFound {
			lines = append(lines, newKeyValueString)
		}

		newFileContent := strings.Join(lines[:], "\n")
		err = ioutil.WriteFile(filename, []byte(newFileContent), 0)
		if err != nil {
			fmt.Printf("Unable to commit the entry to the server log file %s", err)
			return err
		}

		// Once it has been applied to the server file, we can set the commit index to the lastLogEntryIndex (should be kept here in the local variable to avoid conflicts with the parallel requests that might update it?)
		commitIndex = int(math.Max(float64(commitIndex), float64(logEntryIndex)))
		lastAppliedIndex = commitIndex

		return nil
	}
	return errors.New("LeaderIndex:" + strconv.Itoa(leaderIndex))
}

func writeEntryToLogFile(entry string) error {
	file, err := os.OpenFile(config[serverIndex]["logfile"], os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println("Unable to open the leader log file", err)
		return err
	}
	defer file.Close()

	if _, err = file.WriteString(entry); err != nil {
		fmt.Println("Unable to write the get entry to the leader log file", err)
		return err
	}
	return nil
}

// AppendEntries ...
// Heartbeat to inform followers to commit
// Heartbeat to sync followers
// Should appendEntries be able to declare that it's now the leader? -- like the caller passes its own serverId? Its only called by the leader anyway
// Update lastheardfromleader time
// If possible update the timeout time for leader election
// ...
// if leaderIndex != leaderID, change the leaderID index maintained on this server
// entries argument empty for heartbeat
// TODO: @Swati
// func (t *Task) AppendEntries(leaderTerm int, leaderID int, prevLogIndex int, prevLogTerm int, entries []LogEntry, leaderCommitIndex int, currentTerm *int, success *bool) error {
func (t *Task) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReturn) error {
	// if leaderCommitIndex > commitIndex on server, apply the commits to the log file (should only be done after replication and logMatching is successful)
	// if currentTerm > leaderTerm, return false
	// if term at server log prevLogIndex != prevLogTerm, return false
	// if matched, then delete all entries after prevLogIndex, and add the new entries.
	// Update commitIndex = min(last index in log, leaderCommitIndex)

	// read the current term and serverVotedFor from the log file (only have to read the first 2 lines)
	// also read the last line of the logfile to see what the last log index is

	// check if the entries is null, then this is a heartbeat and check for the leader and update the current term and leader if not on track and update the timeout time for leader election

	metadataFile := config[serverIndex]["metadata"]
	logFile := config[serverIndex]["logfile"]
	me.lastMessageTime = time.Now().UnixNano()

	// heartbeat
	if len(args.entries) == 0 {
		commitIndex = int(math.Min(float64(args.leaderCommitIndex), float64(lastLogEntryIndex)))
		if args.leaderTerm > serverCurrentTerm {
			leaderIndex = args.leaderID
			reply.currentTerm = args.leaderTerm
			serverVotedFor = -1
			lines := [2]string{strconv.Itoa(args.leaderTerm), "-1"}
			err := ioutil.WriteFile(metadataFile, []byte(strings.Join(lines[:], "\n")), 0)
			if err != nil {
				fmt.Println("Unable to write the new metadata information", err)
				return err
			}
		}
		return nil
	}

	// return false if recipient's term > leader's term
	if serverCurrentTerm > args.leaderTerm {
		reply.currentTerm = serverCurrentTerm
		reply.success = false
		return nil
	}

	// if a new leader asked us to join their cluster, do so. (this is to stop leader election if this server is candidating an election)
	if args.leaderTerm >= me.currentTerm {
		me.discardElectionCycle = true
		me.currentTerm = args.leaderTerm
	}

	leaderIndex = args.leaderID

	fileContent, err := ioutil.ReadFile(logFile)
	if err != nil {
		fmt.Println("Unable to read the log file", err)
		return err
	}

	lines := strings.Split(string(fileContent), "\n")

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.Split(lines[i], ",")
		logIndex, _ := strconv.Atoi(line[3])
		termIndex, _ := strconv.Atoi(line[2])
		if (logIndex == args.prevLogIndex && termIndex != args.prevLogTerm) || logIndex < args.prevLogIndex {
			reply.success = false
			reply.currentTerm = args.leaderTerm
			return nil
		} else if logIndex == args.prevLogIndex {
			break
		}
	}

	// TODO: remove all log entries after the matched logIndex
	logentrystr := ""
	for i := 0; i < len(args.entries); i++ {
		logentrystr += string("\n" + args.entries[i].Key + "," + args.entries[i].Value + "," + args.entries[i].TermID + "," + args.entries[i].IndexID)
	}
	err = writeEntryToLogFile(logentrystr)
	if err != nil {
		fmt.Println("Unable to replicate the log in appendEntries", err)
		return err
	}

	commitIndex = int(math.Min(float64(args.leaderCommitIndex), float64(lastLogEntryIndex)))
	reply.success = true
	reply.currentTerm = args.leaderTerm
	return nil
}

// need to maintain what was the last term voted for
// candidate term is the term proposed by candidate (current term + 1)
// if receiver's term is greater then, currentTerm is returned and candidate has to update it's term (to be taken care in leader election)
func (t *Task) RequestVote(args RequestVoteArgs, reply *RequestVoteResponse) error {
	// Reply false if candidateTerm < currentTerm : TODO: should this be < or <=
	// If votedFor is null or candidateIndex, and candidate's log is as up-to-date (greater term index, same term greater log index) then grant vote
	// also update the currentTerm to the newly voted term and update the votedFor

	me.responses = append(me.responses, Vote{target: me.id, granted: true})

	return nil
}

// when calling this method use the go func format? -- this would launch it on a separate thread -- not sure how this would be different from the normal operation though (can experiment)
func sendLeaderHeartbeats() error {
	// if current server is the leader, then send AppendEntries heartbeat RPC at idle times to prevent leader election and just after election
	for {
		if serverIndex == leaderIndex {
			for i := 0; i < len(config); i++ {
				if i != serverIndex {
					// send appendentries heartbeat rpc in async and no need to wait for response
					client, err := rpc.DialHTTP("tcp", config[i]["host"]+":"+config[i]["port"])
					if err != nil {
						fmt.Println("Leader unable to create connection with another server ", err)
					} else {
						defer client.Close()
						// Assuming for a leader, commit index = lastappliedindex, values that are not needed for heartbeat are simply passed as -1
						client.Go("Task.AppendEntries", AppendEntriesArgs{leaderTerm: serverCurrentTerm, leaderID: serverIndex,
							prevLogIndex: -1, prevLogTerm: -1, entries: nil, leaderCommitIndex: lastAppliedIndex},
							&AppendEntriesReturn{currentTerm: serverCurrentTerm, success: true}, nil)
					}
				}
			}
		} else {
			return nil
		}
		time.Sleep(time.Duration(leaderHeartBeatDuration) * time.Millisecond)
	}
}

// TODO: @Geetika
func applyCommittedEntries() {
	// at an interval of t ms, if (commitIndex > lastAppliedIndex), then apply entries one by one to the server file (state machine)
	serverFileName := config[serverIndex]["filename"]
	logFileName := config[serverIndex]["logFile"]
	for {
		if lastAppliedIndex < commitIndex {
			lastAppliedIndex++

			fileContent, _ := ioutil.ReadFile(logFileName)
			logs := strings.Split(string(fileContent), "\n")
			currentLog := logs[lastAppliedIndex]
			log := strings.Split(currentLog, ",")
			value := log[1]

			// Add/update key value pair to the server if it is only a put request
			if value != "" {
				fileContent, _ := ioutil.ReadFile(serverFileName)
				lines := strings.Split(string(fileContent), "\n")

				keyFound := false
				for i := 1; i < len(lines); i++ {
					line := strings.Split(lines[i], ",")
					if line[0] == log[0] {
						lines[i] = currentLog
						keyFound = true
						break
					}
				}

				if !keyFound {
					lines = append(lines, currentLog)
				}

				newFileContent := strings.Join(lines[:], "\n")
				err := ioutil.WriteFile(serverFileName, []byte(newFileContent), 0)
				if err != nil {
					fmt.Println("Unable to write to file in applyCommittedEntries", err)
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond) // check after every 100ms
		}
	}
}

// candidate election, request vote
func LeaderElection() error {
	// make this func async
	// in an infinite loop, check if currentTime - lastHeartbeatFromLeader > leaderElectionTimeout, initiate election
	// sleep for leaderElectionTimeout
	// probably need to extend the sleep/waiting time everytime lastHeartbeatfromleader is received (variable value can change in AppendEntries)
	for me.alive {
		me.discardElectionCycle = false
		var lastNap = rand.Intn(me.electionMaxTime-me.electionMinTime) + me.electionMinTime
		var asleep = time.Now().UnixNano()
		time.Sleep(time.Duration(lastNap) * time.Millisecond)

		// no election necessary if the last message was received after we went to sleep.
		if me.lastMessageTime > asleep {
			continue
		}

		/*
		 * oy, it's election time!
		 */
		var electionTimeout = (time.Now().UnixNano()) + int64(rand.Intn(me.electionMaxTime-me.electionMinTime)+me.electionMinTime)
		// For leaderElection: Increment currentTerm
		me.currentTerm++

		// Make RequestVote RPC calls to all other servers. (count its own vote +1) -- Do this async
		// forget the responses we've received so far.
		me.responses = me.responses[:0]

		for _, server := range me.cluster {
			if (server.id == me.id) || ((time.Now().UnixNano()) > electionTimeout) {
				continue
			}

			// TODO: Nick Daly (fix this)
			// go RequestVote(server)
		}

		for (len(me.responses) < len(me.cluster)) && (time.Now().UnixNano() > electionTimeout) {
			time.Sleep(time.Duration(int(math.Min(float64(100), float64(time.Now().UnixNano()-electionTimeout/10)))) * time.Millisecond) // wait a ms...
		}

		// if no leader was elected within the timeout, start a new election cycle
		// or, if we somehow got more votes than the cluster contains, throw out the results.
		if (len(me.responses) < len(me.cluster) && time.Now().UnixNano() > electionTimeout) || (len(me.responses) > len(me.cluster)) || me.discardElectionCycle {
			continue
		}

		// enough votes were collected to tally.
		var myVotes = 0
		// If majority true response received, change leaderIndex = serverIndex
		if len(me.responses) == len(me.cluster) {
			for _, vote := range me.responses {
				if (vote.target == me.id) && vote.granted {
					myVotes++
				}
			}

			// then I won the election.
			if myVotes > (len(me.cluster) / 2) {

				// Reinit entries for matchIndex = 0 and nextIndex = last log index + 1 array for all servers
				lastLogEntryIndex = 0

				// Send AppendEntires Heartbeat RPC to all servers to announce the leadership, also call sendLeaderHeartbeats() asynchronously from here and just let it running
				go sendLeaderHeartbeats()

				// if receiver's term is greater then, currentTerm is returned, update your current term and continue: handled in AppendEntries, line 352.
				return nil
			}
		}
	}

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
// Also update the lastLogEntryIndex, commitIndex updated in appendentries
// TODO: @Geetika
// Doubt: For log replication, we need the logs to follow the same term and index as leader, so changing the entries type to be LogEntry instead of KeyValuePair
// I think, logEntries will depend on next index index of the server, so need to send any logentries in this function as parameter.
func LogReplication() error {
	filePath := config[leaderIndex]["logFile"]
	for index := range config {
		if index != leaderIndex {

			var logEntries []LogEntry

			// prepare logs to send
			fileContent, _ := ioutil.ReadFile(filePath)
			logs := strings.Split(string(fileContent), "\n")
			for j := nextIndex[index]; j < len(logs); j++ {
				log := strings.Split(logs[j], ",")
				le := LogEntry{Key: log[0], Value: log[1], TermID: log[2], IndexID: log[3]}
				logEntries = append(logEntries, le)
			}

			prevlogIndex := nextIndex[index] - 1
			prevlogTerm := strings.Split(logs[prevlogIndex], ",")[2]

			go func(server int, filePath string) {
				prevlogTermInt, err := strconv.Atoi(prevlogTerm)
				appendEntriesArgs := &AppendEntriesArgs{
					leaderTerm:        serverCurrentTerm,
					leaderID:          leaderIndex,
					prevLogIndex:      prevlogIndex,
					prevLogTerm:       prevlogTermInt,
					entries:           logEntries,
					leaderCommitIndex: commitIndex,
				}

				var appendEntriesReturn AppendEntriesReturn

				client, err := rpc.DialHTTP("tcp", config[server]["host"]+":"+config[server]["port"])
				if err != nil {
					fmt.Printf("%s ", err)
				} else {
					defer client.Close()

					err := client.Call("Task.AppendEntries", appendEntriesArgs, &appendEntriesReturn)
					if err == nil {
						if appendEntriesReturn.success {
							matchIndex[server] = prevlogIndex + len(logEntries)
							nextIndex[server] = matchIndex[server] + 1
						} else {
							if appendEntriesReturn.currentTerm > serverCurrentTerm {
								leaderIndex = server // if current term of the server is greater than leader term, the current leader will become the follower.
								// Doubt: though this might not save the actual leader index.
							} else {
								nextIndex[server] = nextIndex[server] - 1

								// next heartbeat request will send the new log entries starting from nextIndex[server] - 1
							}
						}
					} else {
						fmt.Printf("%s ", err)
					}

					fileContent, err := ioutil.ReadFile(filePath)

					if err != nil {
						fmt.Println(err)
					}

					logs := strings.Split(string(fileContent), "\n")

					for N := len(logs) - 1; N > commitIndex; N-- {
						log := strings.Split(logs[N], ",")
						count := 0
						logTerm, _ := strconv.Atoi(log[2])
						if logTerm == serverCurrentTerm {
							for i := range config {
								if i != leaderIndex {
									if matchIndex[i] >= N {
										count++
									}
								}
							}
						}
						if count >= majoritySize {
							commitIndex = N
							break
						}
					}
				}
			}(index, filePath)
		}
	}
	return nil
}

//Init ... takes in config and index of the current server in config
func Init(index int) error {
	t := new(Task)

	err := rpc.Register(t)

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
	log.Printf("Serving RPC server on port %s", config[index]["port"])

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
	// TODO: initialize the leaderElectionTimeout time randomly selected between timeoutmin and timeoutmax, initialise the last log entry index, current term, serverVotedFor etc by reading from persistent storage
	pid := os.Getpid()
	fmt.Printf("Server %d starts with process id: %d\n", serverIndex, pid)
	filename := config[serverIndex]["filename"]
	logFileName := config[serverIndex]["logfile"]
	metadataFileName := config[serverIndex]["metadata"]

	_, err := os.Stat(filename)

	// TODO: Create server file and log file if not already present
	if os.IsNotExist(err) {
		_, err := os.OpenFile(filename, os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create file ", err)
		}
	}

	_, err = os.Stat(logFileName)
	if os.IsNotExist(err) {
		_, err := os.OpenFile(logFileName, os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create log file ", err)
		}
	}

	_, err = os.Stat(metadataFileName)
	if os.IsNotExist(err) {
		_, err := os.OpenFile(metadataFileName, os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create meta data file ", err)
		}
	}

	go LeaderElection()
	go applyCommittedEntries() // running this over a separate thread for indefinite time
	me := RaftServer{alive: true}
	me.responses = make([]Vote, 10)
	err = Init(serverIndex)
}
