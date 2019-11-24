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

// TODO: @Swati: add these 2 variables (serverCurrentTerm and serverVotedFor) in the RaftServer struct
// Persistent state on all servers: currentTerm, votedFor (candidate ID that received vote in current term)-- using a separate metadata file for this
// var serverVotedFor int --- These 2 have to be added in logFile (may be use the first 2 lines of the logFile)
// var currentTerm int (Also maintaining it in volatile memory to avoid Disk IO everytime)
// var me.currentTerm int // start value could be 0 (init from persistent storage during start)
// var serverVotedFor int // start value could be -1 (init from persistent storage during start)

var me RaftServer

type RaftServer struct {
	// who am I?
	serverIndex int
	alive       bool

	// when is it?
	currentTerm       int   // start value could be 0 (init from persistent storage during start)
	serverVotedFor    int   // where is this even being used? -- TODO: check this, May be separate variable defined for this
	lastMessageTime   int64 // time in nanoseconds
	responses         []Vote
	commitIndex       int // DOUBT: index of the last committed entry (should these 2 be added to persistent storage as well? How are we going to tell till where is the entry applied in the log?)
	lastAppliedIndex  int
	lastLogEntryIndex int //TODO: init from persistent storage on start

	// elections
	leaderIndex             int
	electionMinTime         int // time in nanoseconds // TODO: need to init this, May be check the timeout size effect on the overall system performance
	electionMaxTime         int // time in nanoseconds
	discardElectionCycle    bool
	leaderHeartBeatDuration int // TODO: init this (time in millisecond)

	// Log replication -- structures needed at leader
	// would be init in leader election
	nextIndex  [numServers]int // this is to define which values to send for replication to each server during AppendEntries (initialize to last log index on leader + 1)
	matchIndex [numServers]int // not sure where this is used -- but keeping it for now (initialize to 0)
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
	LeaderTerm, LeaderID, PrevLogIndex, PrevLogTerm int
	Entries                                         []LogEntry
	LeaderCommitIndex                               int
}

type AppendEntriesReturn struct {
	CurrentTerm int
	Success     bool
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
	if me.serverIndex == me.leaderIndex {
		// read the value of the key from the file and return
		// do log replication and commit on get entries to get the most consistent response in case of leader failure

		// first add the entry to the log of the leader, then call logReplication, it returns when replicated to majority
		// once returned by logreplication, commit to file and return success, also update the commitIndex and lastappliedindex

		me.lastLogEntryIndex++                // what about 2 parallel requests trying to use the same logEntryIndex (this is a shared variable -- do we need locks here?)
		logEntryIndex := me.lastLogEntryIndex // these 2 steps should be atomic

		logentrystr := string("\n" + key + ",," + strconv.Itoa(me.currentTerm) + "," + strconv.Itoa(logEntryIndex))

		if logEntryIndex == 1 {
			logentrystr = string(key + ",," + strconv.Itoa(me.currentTerm) + "," + strconv.Itoa(logEntryIndex))
		}

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
		me.commitIndex = int(math.Max(float64(me.commitIndex), float64(logEntryIndex)))
		me.lastAppliedIndex = me.commitIndex
		// fetch the value of the key

		file, err := os.Open(config[me.serverIndex]["filename"])
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
	return errors.New("LeaderIndex:" + strconv.Itoa(me.leaderIndex))
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
	if me.serverIndex == me.leaderIndex {
		me.lastLogEntryIndex++                // what about 2 parallel requests trying to use the same logEntryIndex (this is a shared variable -- do we need locks here?)
		logEntryIndex := me.lastLogEntryIndex // these 2 steps should be atomic

		logentrystr := string("\n" + keyValue.Key + "," + keyValue.Value + "," + strconv.Itoa(me.currentTerm) + "," + strconv.Itoa(logEntryIndex))
		if logEntryIndex == 1 {
			logentrystr = string(keyValue.Key + "," + keyValue.Value + "," + strconv.Itoa(me.currentTerm) + "," + strconv.Itoa(logEntryIndex))
		}

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
		filename := config[me.serverIndex]["filename"]
		fileContent, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Println(err)
			return err
		}

		lines := []string{}
		if len(string(fileContent)) != 0 {
			lines = strings.Split(string(fileContent), "\n")
		}

		for i := 0; i < len(lines); i++ {
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
		me.commitIndex = int(math.Max(float64(me.commitIndex), float64(logEntryIndex)))
		me.lastAppliedIndex = me.commitIndex
		return nil
	}
	return errors.New("LeaderIndex:" + strconv.Itoa(me.leaderIndex))
}

func writeEntryToLogFile(entry string) error {
	file, err := os.OpenFile(config[me.serverIndex]["logfile"], os.O_APPEND|os.O_WRONLY, 0600)
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

	fmt.Printf("append entries calling \n")
	metadataFile := config[me.serverIndex]["metadata"]
	logFile := config[me.serverIndex]["logfile"]
	me.lastMessageTime = time.Now().UnixNano()

	fmt.Printf("1 \n")

	// heartbeat
	if len(args.Entries) == 0 {
		fmt.Printf("2 \n")
		me.commitIndex = int(math.Min(float64(args.LeaderCommitIndex), float64(me.lastLogEntryIndex)))
		if args.LeaderTerm > me.currentTerm {
			me.leaderIndex = args.LeaderID
			reply.CurrentTerm = args.LeaderTerm
			me.serverVotedFor = -1
			lines := [2]string{strconv.Itoa(args.LeaderTerm), "-1"}
			err := ioutil.WriteFile(metadataFile, []byte(strings.Join(lines[:], "\n")), 0)
			if err != nil {
				fmt.Println("Unable to write the new metadata information", err)
				return err
			}
		}
		fmt.Printf("3 \n")
		return nil
	}

	fmt.Printf("4 \n")

	// return false if recipient's term > leader's term
	if me.currentTerm > args.LeaderTerm {
		reply.CurrentTerm = me.currentTerm
		reply.Success = false
		return nil
	}
	fmt.Printf("5 \n")

	// if a new leader asked us to join their cluster, do so. (this is to stop leader election if this server is candidating an election)
	if args.LeaderTerm >= me.currentTerm {
		me.discardElectionCycle = true
		me.currentTerm = args.LeaderTerm
	}

	fmt.Printf("6 \n")

	me.leaderIndex = args.LeaderID

	fileContent, err := ioutil.ReadFile(logFile)
	if err != nil {
		fmt.Println("Unable to read the log file", err)
		return err
	}

	lines := []string{}
	if len(string(fileContent)) != 0 {
		lines = strings.Split(string(fileContent), "\n")
	}

	fmt.Printf("7 \n")

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.Split(lines[i], ",")
		logIndex, _ := strconv.Atoi(line[3])
		termIndex, _ := strconv.Atoi(line[2])
		if (logIndex == args.PrevLogIndex && termIndex != args.PrevLogTerm) || logIndex < args.PrevLogIndex {
			reply.Success = false
			reply.CurrentTerm = args.LeaderTerm
			return nil
		} else if logIndex == args.PrevLogIndex {
			break
		}
	}
	fmt.Printf("8 \n")
	// TODO: remove all log entries from lines after the matched logIndex
	// logentrystr := lines.Join("\n")
	logsToAppend := []string{}
	for i := 0; i < len(args.Entries); i++ {
		logsToAppend = append(logsToAppend, string(args.Entries[i].Key+","+args.Entries[i].Value+","+args.Entries[i].TermID+","+args.Entries[i].IndexID))
	}
	logsToAppendStr := strings.Join(logsToAppend, "\n")
	if args.PrevLogIndex >= 0 {
		logsToAppendStr = "\n" + logsToAppendStr
	}
	err = writeEntryToLogFile(logsToAppendStr)
	if err != nil {
		fmt.Println("Unable to replicate the log in appendEntries", err)
		return err
	}
	fmt.Printf("9 \n")
	me.lastLogEntryIndex, _ = strconv.Atoi(args.Entries[len(args.Entries)-1].IndexID)
	me.commitIndex = int(math.Min(float64(args.LeaderCommitIndex), float64(me.lastLogEntryIndex)))
	fmt.Println("commit index ", args.LeaderCommitIndex, me.commitIndex)
	reply.Success = true
	reply.CurrentTerm = args.LeaderTerm

	fmt.Printf("append entries done \n")

	return nil
}

// need to maintain what was the last term voted for
// candidate term is the term proposed by candidate (current term + 1)
// if receiver's term is greater then, currentTerm is returned and candidate has to update it's term (to be taken care in leader election)
func (t *Task) RequestVote(args RequestVoteArgs, reply *RequestVoteResponse) error {
	// Reply false if candidateTerm < currentTerm : TODO: should this be < or <=
	// If votedFor is null or candidateIndex, and candidate's log is as up-to-date (greater term index, same term greater log index) then grant vote
	// also update the currentTerm to the newly voted term and update the votedFor

	me.responses = append(me.responses, Vote{target: me.serverIndex, granted: true})

	return nil
}

// when calling this method use the go func format? -- this would launch it on a separate thread -- not sure how this would be different from the normal operation though (can experiment)
func sendLeaderHeartbeats() error {
	// if current server is the leader, then send AppendEntries heartbeat RPC at idle times to prevent leader election and just after election
	for {
		if me.serverIndex == me.leaderIndex {
			for i := 0; i < len(config); i++ {
				if i != me.serverIndex {
					// send appendentries heartbeat rpc in async and no need to wait for response
					client, err := rpc.DialHTTP("tcp", config[i]["host"]+":"+config[i]["port"])
					if err != nil {
						fmt.Println("Leader unable to create connection with another server ", err)
					} else {
						defer client.Close()
						// Assuming for a leader, commit index = lastappliedindex, values that are not needed for heartbeat are simply passed as -1
						client.Go("Task.AppendEntries", AppendEntriesArgs{LeaderTerm: me.currentTerm, LeaderID: me.serverIndex,
							PrevLogIndex: -1, PrevLogTerm: -1, Entries: nil, LeaderCommitIndex: me.lastAppliedIndex},
							&AppendEntriesReturn{CurrentTerm: me.currentTerm, Success: true}, nil)
					}
				}
			}
		} else {
			return nil
		}
		time.Sleep(time.Duration(me.leaderHeartBeatDuration) * time.Millisecond)
	}
}

// TODO: @Geetika
func applyCommittedEntries() {
	// at an interval of t ms, if (commitIndex > lastAppliedIndex), then apply entries one by one to the server file (state machine)
	serverFileName := config[me.serverIndex]["filename"]
	logFileName := config[me.serverIndex]["logfile"]
	for {
		if me.lastAppliedIndex < me.commitIndex {
			me.lastAppliedIndex++

			fileContent, _ := ioutil.ReadFile(logFileName)
			logs := strings.Split(string(fileContent), "\n")
			currentLog := logs[me.lastAppliedIndex]
			log := strings.Split(currentLog, ",")
			currentEntryStr := log[0] + "," + log[1]
			fmt.Println("log is ", log, currentLog)
			value := log[1]
			fmt.Println("value is ", value, currentLog, log)

			// Add/update key value pair to the server if it is only a put request
			if value != "" {
				fileContent, _ := ioutil.ReadFile(serverFileName)
				lines := []string{}
				if len(string(fileContent)) != 0 {
					lines = strings.Split(string(fileContent), "\n")
				}

				keyFound := false
				for i := 1; i < len(lines); i++ {
					line := strings.Split(lines[i], ",")
					if line[0] == log[0] {
						lines[i] = currentEntryStr
						keyFound = true
						break
					}
				}

				if !keyFound {
					lines = append(lines, currentEntryStr)
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
	// in an infinite loop, check if currentTime - me.lastMessageTime > electionTimeout, initiate election
	// sleep for electionTimeout
	// probably need to extend the sleep/waiting time everytime me.lastMessageTime is received (variable value can change in AppendEntries)
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

		for index := range config {
			if (index == me.serverIndex) || ((time.Now().UnixNano()) > electionTimeout) {
				continue
			}

			// TODO: Nick Daly (fix this)
			// go RequestVote(server)
		}

		for (len(me.responses) < numServers) && (time.Now().UnixNano() > electionTimeout) {
			time.Sleep(time.Duration(int(math.Min(float64(100), float64(time.Now().UnixNano()-electionTimeout/10)))) * time.Millisecond) // wait a ms...
		}

		// if no leader was elected within the timeout, start a new election cycle
		// or, if we somehow got more votes than the cluster contains, throw out the results.
		if (len(me.responses) < numServers && time.Now().UnixNano() > electionTimeout) || (len(me.responses) > numServers) || me.discardElectionCycle {
			continue
		}

		// enough votes were collected to tally.
		var myVotes = 0
		// If majority true response received, change leaderIndex = serverIndex
		if len(me.responses) == numServers {
			for _, vote := range me.responses {
				if (vote.target == me.serverIndex) && vote.granted {
					myVotes++
				}
			}

			// then I won the election.
			if myVotes > majoritySize {

				// Reinit entries for matchIndex = 0 and nextIndex = last log index + 1 array for all servers
				me.lastLogEntryIndex = 0 // TODO: WHAT??? -- This is probably an error -- Wrong variable used

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
	fmt.Printf("LogReplication calling\n")
	filePath := config[me.leaderIndex]["logfile"]
	for index := range config {
		if index != me.leaderIndex {

			var logEntries []LogEntry

			// prepare logs to send
			fileContent, err := ioutil.ReadFile(filePath)
			if err != nil {
				fmt.Printf("error %s\n", err)
			}

			logs := strings.Split(string(fileContent), "\n")
			for j := me.nextIndex[index]; j < len(logs); j++ {
				log := strings.Split(logs[j], ",")
				le := LogEntry{Key: log[0], Value: log[1], TermID: log[2], IndexID: log[3]}
				logEntries = append(logEntries, le)
			}
			fmt.Println(index, logEntries)

			prevlogIndex := me.nextIndex[index] - 1
			prevlogTerm := "0"
			if prevlogIndex != -1 {
				prevlogTerm = strings.Split(logs[prevlogIndex], ",")[2]
			}

			server := index
			// go func(server int, filePath string) {
			prevlogTermInt, _ := strconv.Atoi(prevlogTerm)
			appendEntriesArgs := &AppendEntriesArgs{
				LeaderTerm:        me.currentTerm,
				LeaderID:          me.leaderIndex,
				PrevLogIndex:      prevlogIndex,
				PrevLogTerm:       prevlogTermInt,
				Entries:           logEntries,
				LeaderCommitIndex: me.commitIndex,
			}

			var appendEntriesReturn AppendEntriesReturn

			client, err := rpc.DialHTTP("tcp", config[server]["host"]+":"+config[server]["port"])
			if err != nil {
				fmt.Printf("%s ", err)
			} else {
				defer client.Close()

				err = client.Call("Task.AppendEntries", appendEntriesArgs, &appendEntriesReturn)
				if err == nil {
					if appendEntriesReturn.Success {
						me.matchIndex[server] = prevlogIndex + len(logEntries) - 1
						me.nextIndex[server] = me.matchIndex[server] + 1
					} else {
						if appendEntriesReturn.CurrentTerm > me.currentTerm {
							me.leaderIndex = server // if current term of the server is greater than leader term, the current leader will become the follower.
							// Doubt: though this might not save the actual leader index.
						} else {
							me.nextIndex[server] = me.nextIndex[server] - 1

							// next heartbeat request will send the new log entries starting from nextIndex[server] - 1
						}
					}
				} else {
					fmt.Printf("error from append entry \n")
					fmt.Printf("%s ", err)
				}

				fileContent, err := ioutil.ReadFile(filePath)

				if err != nil {
					fmt.Println(err)
				}

				logs := strings.Split(string(fileContent), "\n")

				for N := len(logs) - 1; N > me.commitIndex; N-- {
					log := strings.Split(logs[N], ",")
					count := 0
					logTerm, _ := strconv.Atoi(log[2])
					if logTerm == me.currentTerm {
						for i := range config {
							if i != me.leaderIndex {
								if me.matchIndex[i] >= N {
									count++
								}
							}
						}
					}
					if count >= majoritySize {
						me.commitIndex = N
						break
					}
				}
			}
			// }(index, filePath)
		}
	}
	return nil
}

//Init ... takes in config and index of the current server in config
func Init(index int) error {
	t := new(Task)
	err := rpc.Register(t)

	me.serverIndex = index
	me.leaderIndex = 0
	me.alive = true

	// TODO: can keep the lastAppliedIndex and commitIndex in metadata files
	me.lastAppliedIndex = -1
	me.commitIndex = -1

	me.nextIndex[0] = 0
	me.matchIndex[0] = 0

	fileContent, _ := ioutil.ReadFile(config[me.leaderIndex]["logfile"])
	lines := []string{}
	if len(string(fileContent)) != 0 {
		lines = strings.Split(string(fileContent), "\n")
	}

	if me.serverIndex != me.leaderIndex {
		me.nextIndex[me.serverIndex] = len(lines)
		me.matchIndex[me.serverIndex] = 0
	}
	me.lastLogEntryIndex = len(lines)

	// TODO: call leader election function asynchronously so that it's non blocking
	// call applyCommittedEntries asynchronously so that it keeps running in the background
	// go LeaderElection()
	// me.responses = make([]Vote, 10)
	go applyCommittedEntries() // running this over a separate thread for indefinite time

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
	return nil
}

func main() {
	args := os.Args[1:]
	serverIndex, _ := strconv.Atoi(args[0])

	// TODO: initialise the last log entry index, current term, serverVotedFor etc by reading from persistent storage
	pid := os.Getpid()
	fmt.Printf("Server %d starts with process id: %d\n", serverIndex, pid)

	// TODO: Create server file and log file if not already present
	filename := config[serverIndex]["filename"]
	logFileName := config[serverIndex]["logfile"]
	metadataFileName := config[serverIndex]["metadata"]

	_, err := os.Stat(filename)
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

	err = Init(serverIndex)
}
