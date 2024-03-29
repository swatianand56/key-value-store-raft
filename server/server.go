// see the key-value-distributed to see how rpcs are implemented.
// nick: Leader election
// nick: request vote
// append entries
// consistency checks
// client input
// unit tests

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var activeServerFilename = "./activeServers.cfg"

// Make this dynamic initialization based on numServers

type Config struct {
	Servers []ServerConfig `json:"servers"`
}

type ServerConfig struct {
	Port     string `json:"port"`
	Host     string `json:"host"`
	Filename string `json:"filename"`
	LogFile  string `json:"logfile"`
	Metadata string `json:"metadata"`
}

var config Config

// Persistent state on all servers: currentTerm, votedFor (candidate ID that received vote in current term)-- using a separate metadata file for this
// var currentTerm int (Also maintaining it in volatile memory to avoid Disk IO everytime)
// var me.currentTerm int // start value could be 0 (init from persistent storage during start)

var me RaftServer

//GetKey ... Leader only servers this request always
// If this server not the leader, redirect the request to the leader (How can client library know who the leader is?)
func (me *RaftServer) GetKey(key string, value *string) error {
	if me.serverIndex == me.leaderIndex {
		// read the value of the key from the file and return
		// do log replication and commit on get entries to get the most consistent response in case of leader failure

		// first add the entry to the log of the leader, then call logReplication, it returns when replicated to majority
		// once returned by logreplication, commit to file and return success, also update the commitIndex and lastappliedindex

		me.mux.Lock()
		me.lastLogEntryIndex++                // what about 2 parallel requests trying to use the same logEntryIndex (this is a shared variable -- do we need locks here?)
		logEntryIndex := me.lastLogEntryIndex // these 2 steps should be atomic
		writeLogEntryInMemory(LogEntry{Key: key, Value: "", TermID: strconv.Itoa(me.currentTerm), IndexID: strconv.Itoa(logEntryIndex)})
		logs := me.logs
		me.mux.Unlock()

		err := writeEntryToLogFile(logs)
		if err != nil {
			return err
		}

		err = LogReplication(logEntryIndex)
		if err != nil {
			fmt.Println("Unable to replicate the get request to a majority of servers", err)
			return err
		}
		me.mux.Lock()
		// if successful, apply log entry to the file (fetch the value of the key from the file for get)
		// TODO: how to ensure order in which entries are committed -- can index 12 commit before 11 is committed
		// What's the possibility of using No-op here?
		me.commitIndex = int(math.Max(float64(me.commitIndex), float64(logEntryIndex)))
		for i := me.lastAppliedIndex; i < me.commitIndex; i++ {
			applyEntries()
		}
		// fetch the value of the key
		data := me.data
		me.mux.Unlock()

		for i := 0; i < (len(data)); i++ {
			line := data[i]
			if line.Key == key {
				*value = line.Value
				return nil
			}
		}

		return nil
	}
	// cannot find a way to transfer connection to another server, so throwing error with leaderIndex
	return errors.New("LeaderIndex:" + strconv.Itoa(me.leaderIndex))
}

//PutKey ...
// The request should only come to the leader. If this is not the leader redirect the request to the leader
func (me *RaftServer) PutKey(keyValue KeyValuePair, oldValue *string) error {
	// Make entry in the log file
	// Call the LogReplication function and take care of steps 2,3,4 there
	// 2. Send AppendEntries to all other servers in async
	// 3. When majority response received, add to the server file
	// 4. If AppendEntries fail, need to update the lastMatchedLogIndex for that server
	// Keep trying indefinitely unless the entry is replicated on all servers
	// Update the last commit ID? -- Do we need it as a variable?, replicate the data in log and lastAppliedIndex
	// return response to the client
	if me.serverIndex == me.leaderIndex {
		me.mux.Lock()
		me.lastLogEntryIndex++                // what about 2 parallel requests trying to use the same logEntryIndex (this is a shared variable -- do we need locks here?)
		logEntryIndex := me.lastLogEntryIndex // these 2 steps should be atomic
		writeLogEntryInMemory(LogEntry{Key: keyValue.Key, Value: keyValue.Value, TermID: strconv.Itoa(me.currentTerm), IndexID: strconv.Itoa(logEntryIndex)})
		logs := me.logs
		me.mux.Unlock()

		err := writeEntryToLogFile(logs)
		if err != nil {
			return err
		}
		// what if failure happens, do we decrement the lastLogEntryIndex? what if another request has updated it already -- there might be a conflict here
		err = LogReplication(logEntryIndex)
		if err != nil {
			fmt.Println("Unable to replicate the get request to a majority of servers", err)
			return err
		}

		// Written to log, now do the operation on the file (following write-ahead logging semantics)
		me.mux.Lock()
		// Once it has been applied to the server file, we can set the commit index to the lastLogEntryIndex (should be kept here in the local variable to avoid conflicts with the parallel requests that might update it?)
		me.commitIndex = int(math.Max(float64(me.commitIndex), float64(logEntryIndex)))
		for i := me.lastAppliedIndex; i < me.commitIndex; i++ {
			applyEntries()
		}
		me.mux.Unlock()

		if err != nil {
			return err
		}

		return nil
	}
	// fmt.Println("throwing error to putkey request ", me.leaderIndex)
	return errors.New("LeaderIndex:" + strconv.Itoa(me.leaderIndex))
}

func (me *RaftServer) ChangeMembership(newConfig []int, reply *int) error {
	// create cold, new config (log replicate)
	// new servers ()
	me.mux.Lock()
	me.lastLogEntryIndex = me.lastLogEntryIndex + 1
	logEntryIndex := me.lastLogEntryIndex
	newConfigStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(newConfig)), ","), "[]")
	le := LogEntry{Key: "config-old-new", Value: newConfigStr, TermID: strconv.Itoa(me.currentTerm), IndexID: strconv.Itoa(logEntryIndex)}
	writeLogEntryInMemory(le)
	me.newServers = newConfig
	for _, i := range newConfig {
		if !isElementPresentInArray(me.activeServers, i) {
			me.nextIndex[i] = 0
			me.matchIndex[i] = 0
		}
	}
	logs := me.logs
	me.mux.Unlock()

	err := writeEntryToLogFile(logs)

	if err != nil {
		fmt.Println("err in config change", err)
		me.newServers = make([]int, 0)
		return err
	}

	err = LogReplication(logEntryIndex)

	if err != nil {
		fmt.Println("unable to replicate config changes to majority servers", err)
		me.newServers = make([]int, 0)
		return err
	}

	me.mux.Lock()
	me.commitIndex = int(math.Max(float64(me.commitIndex), float64(logEntryIndex)))
	for i := me.lastAppliedIndex; i < me.commitIndex; i++ {
		applyEntries()
	}

	me.lastLogEntryIndex = me.lastLogEntryIndex + 1
	logEntryIndex = me.lastLogEntryIndex
	le = LogEntry{Key: "config-new", Value: newConfigStr, TermID: strconv.Itoa(me.currentTerm), IndexID: strconv.Itoa(logEntryIndex)}
	writeLogEntryInMemory(le)
	logs = me.logs
	me.mux.Unlock()

	err = writeEntryToLogFile(logs)

	if err != nil {
		fmt.Println("err in new config change", err)
		me.newServers = make([]int, 0)
		return err
	}

	err = LogReplication(logEntryIndex)

	if err != nil {
		fmt.Println("unable to replicate config new to majority servers", err)
		me.newServers = make([]int, 0)
		return err
	}

	me.mux.Lock()
	// ToDo - Not sure whether to wait for consensus or not?
	me.activeServers = me.newServers
	me.newServers = make([]int, 0)
	me.commitIndex = int(math.Max(float64(me.commitIndex), float64(logEntryIndex)))
	for i := me.lastAppliedIndex; i < me.commitIndex; i++ {
		applyEntries()
	}
	//TODO: step down the leader if not in new config
	me.mux.Unlock()

	err = ioutil.WriteFile(activeServerFilename, []byte(newConfigStr), 0)

	if !isElementPresentInArray(me.activeServers, me.serverIndex) {
		me.leaderIndex = -1
	}

	if err != nil {
		fmt.Println("could not write final config change to the server active servers file")
	}

	return nil
}

func writeDataInFile(totalData []KeyValuePair) error {
	lines := []string{}
	for _, data := range totalData {
		datastr := data.Key + "," + data.Value
		lines = append(lines, datastr)
	}
	err := ioutil.WriteFile(config.Servers[me.serverIndex].Filename, []byte(strings.Join(lines[:], "\n")), 0)

	return err
}

func writeEntryToLogFile(logs []LogEntry) error {
	lines := []string{}
	for _, log := range logs {
		logstr := log.Key + "," + log.Value + "," + log.TermID + "," + log.IndexID
		lines = append(lines, logstr)
	}
	err := ioutil.WriteFile(config.Servers[me.serverIndex].LogFile, []byte(strings.Join(lines[:], "\n")), 0)

	return err
}

func writeLogEntryInMemory(entry LogEntry) {
	me.logs = append(me.logs, entry)
	me.lastLogEntryIndex, _ = strconv.Atoi(entry.IndexID)
	me.lastLogEntryTerm, _ = strconv.Atoi(entry.TermID)
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
// func (t *Task) AppendEntries(leaderTerm int, leaderID int, prevLogIndex int, prevLogTerm int, entries []LogEntry, leaderCommitIndex int, currentTerm *int, success *bool) error {
func (me *RaftServer) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReturn) error {
	// if leaderCommitIndex > commitIndex on server, apply the commits to the log file (should only be done after replication and logMatching is successful)
	// if currentTerm > leaderTerm, return false
	// if term at server log prevLogIndex != prevLogTerm, return false
	// if matched, then delete all entries after prevLogIndex, and add the new entries.
	// Update commitIndex = min(last index in log, leaderCommitIndex)

	// read the current term and serverVotedFor from the log file (only have to read the first 2 lines)
	// also read the last line of the logfile to see what the last log index is

	// check if the entries is null, then this is a heartbeat and check for the leader and update the current term and leader if not on track and update the timeout time for leader election
	metadataFile := config.Servers[me.serverIndex].Metadata

	me.mux.Lock()
	defer me.mux.Unlock()
	me.lastMessageTime = time.Now().UnixNano()

	// heartbeat
	if len(args.Entries) == 0 {
		reply.CurrentTerm = args.LeaderTerm
		me.commitIndex = int(math.Min(float64(args.LeaderCommitIndex), float64(me.lastLogEntryIndex)))
		if args.LeaderTerm > me.currentTerm {
			me.leaderIndex = args.LeaderID
			me.currentTerm = args.LeaderTerm
			me.serverVotedFor = args.LeaderID
			lines := [2]string{strconv.Itoa(args.LeaderTerm), strconv.Itoa(me.serverVotedFor)}
			err := ioutil.WriteFile(metadataFile, []byte(strings.Join(lines[:], "\n")), 0)
			if err != nil {
				fmt.Println("Unable to write the new metadata information", err)
				return err
			}
		} else if args.LeaderTerm == me.currentTerm && me.leaderIndex == -1 {
			me.leaderIndex = args.LeaderID
		} else if args.LeaderTerm < me.currentTerm {
			reply.CurrentTerm = me.currentTerm
		}
		return nil
	}

	// return false if recipient's term > leader's term
	if me.currentTerm > args.LeaderTerm {
		reply.CurrentTerm = me.currentTerm
		reply.Success = false
		return nil
	}

	// if a new leader asked us to join their cluster, do so. (this is to stop leader election if this server is candidating an election)
	me.leaderIndex = args.LeaderID
	if me.currentTerm != args.LeaderTerm {
		me.currentTerm = args.LeaderTerm
		me.serverVotedFor = -1
		metadataContent := [2]string{strconv.Itoa(me.currentTerm), "-1"}
		err := ioutil.WriteFile(metadataFile, []byte(strings.Join(metadataContent[:], "\n")), 0)
		if err != nil {
			fmt.Println("Unable to write the new metadata information", err)
		}
	}

	for i := len(me.logs) - 1; i >= 0; i-- {
		log := me.logs[i]
		logIndex, _ := strconv.Atoi(log.IndexID)
		termIndex, _ := strconv.Atoi(log.TermID)
		if (logIndex == args.PrevLogIndex && termIndex != args.PrevLogTerm) || logIndex < args.PrevLogIndex {
			reply.Success = false
			reply.CurrentTerm = args.LeaderTerm
			return nil
		} else if logIndex == args.PrevLogIndex {
			break
		}
	}

	me.logs = me.logs[:args.PrevLogIndex+1]

	for i := 0; i < len(args.Entries); i++ {
		writeLogEntryInMemory(args.Entries[i])
	}

	me.lastLogEntryIndex, _ = strconv.Atoi(args.Entries[len(args.Entries)-1].IndexID)
	me.commitIndex = int(math.Min(float64(args.LeaderCommitIndex), float64(me.lastLogEntryIndex)))

	// me.mux.Unlock()

	err := writeEntryToLogFile(me.logs)
	if err != nil {
		fmt.Println("Unable to replicate the log in appendEntries", err)
		reply.Success = false
		reply.CurrentTerm = args.LeaderTerm
		return err
	}

	for i := 0; i < len(args.Entries); i++ {
		key := args.Entries[i].Key
		value := args.Entries[i].Value
		if key == "config-old-new" {
			me.newServers = strToIntArr(strings.Split(value, ","))
		} else if key == "config-new" {
			me.activeServers = me.newServers
			me.newServers = make([]int, 0)
		}
	}

	reply.Success = true
	reply.CurrentTerm = args.LeaderTerm

	return nil
}

// need to maintain what was the last term voted for
// candidate term is the term proposed by candidate (current term + 1)
// if receiver's term is greater then, currentTerm is returned and candidate has to update it's term (to be taken care in leader election)
func (me *RaftServer) RequestVote(args RequestVoteArgs, reply *RequestVoteResponse) error {
	me.mux.Lock()
	allServers := serversUnion(me.newServers, me.activeServers)
	me.mux.Unlock()
	if me.lastMessageTime+int64(me.electionMinTime) > time.Now().UnixNano() || !isElementPresentInArray(allServers, args.CandidateIndex) {
		reply.VoteGranted = false
		return nil
	}
	me.mux.Lock()
	mycurrentTerm := me.currentTerm
	myserverVotedFor := me.serverVotedFor
	mylastLogEntryIndex := me.lastLogEntryIndex
	mylastLogEntryTerm := me.lastLogEntryTerm
	myserverIndex := me.serverIndex
	me.mux.Unlock()

	// Reply false if candidateTerm < currentTerm
	if args.CandidateTerm < mycurrentTerm {
		reply.CurrentTerm = mycurrentTerm
		reply.VoteGranted = false
		return nil
	}

	// If votedFor is null or candidateIndex, and candidate's log is as up-to-date (greater term index, same term greater log index) then grant vote
	// also update the currentTerm to the newly voted term and update the votedFor
	if args.CandidateTerm > mycurrentTerm || myserverVotedFor == -1 || myserverVotedFor == args.CandidateIndex {
		if args.LastLogTerm > mylastLogEntryTerm || (args.LastLogTerm == mylastLogEntryTerm && args.LastLogIndex >= mylastLogEntryIndex) {
			// if I'm outdated, remote has my vote.
			me.mux.Lock()
			me.serverVotedFor = args.CandidateIndex
			me.currentTerm = args.CandidateTerm
			me.leaderIndex = -1
			me.mux.Unlock()
			reply.CurrentTerm = args.CandidateTerm
			reply.VoteGranted = true
			metadataContent := [2]string{strconv.Itoa(args.CandidateTerm), strconv.Itoa(args.CandidateIndex)}
			err := ioutil.WriteFile(config.Servers[myserverIndex].Metadata, []byte(strings.Join(metadataContent[:], "\n")), 0)
			if err != nil {
				fmt.Println("Request Vote RPC: Unable to write to metadata file ", myserverIndex, mycurrentTerm, args.CandidateIndex)
			}
		}
	} else {
		reply.CurrentTerm = mycurrentTerm
		reply.VoteGranted = false
	}

	return nil
}

func (me *RaftServer) Sleep(ms time.Duration, slept *time.Duration) error {
	// sleep for a while, then reply with how long we slept.
	start := time.Now()

	time.Sleep(ms * time.Millisecond)

	*slept = time.Since(start)

	return nil
}

func (me *RaftServer) CurrentState(input int, state *RaftServer) error {
	// copies state from the server to return it for inspection.

	state.serverIndex = me.serverIndex // just in case I get *really* confused.
	state.currentTerm = me.currentTerm
	state.serverVotedFor = me.serverVotedFor
	state.lastMessageTime = me.lastMessageTime
	state.commitIndex = me.commitIndex
	state.lastAppliedIndex = me.lastAppliedIndex
	state.lastLogEntryIndex = me.lastLogEntryIndex
	state.lastLogEntryTerm = me.lastLogEntryTerm
	state.leaderIndex = me.leaderIndex

	return nil
}

func sendHeartbeat(i int) {
	// send appendentries heartbeat rpc in async and no need to wait for response
	conn, err := net.DialTimeout("tcp", config.Servers[i].Host+":"+config.Servers[i].Port, 150*time.Millisecond)
	if err != nil {
		fmt.Println("Leader unable to create connection with another server ", err)
	} else {
		conn.SetDeadline(time.Now().Add(150 * time.Millisecond))
		client := rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		var appendEntriesResult AppendEntriesReturn

		me.mux.Lock()
		mycurrentTerm := me.currentTerm
		myserverIndex := me.serverIndex
		mylastAppliedIndex := me.lastAppliedIndex
		me.mux.Unlock()
		client.Call("RaftServer.AppendEntries", AppendEntriesArgs{LeaderTerm: mycurrentTerm, LeaderID: myserverIndex,
			PrevLogIndex: -1, PrevLogTerm: -1, Entries: nil, LeaderCommitIndex: mylastAppliedIndex},
			&appendEntriesResult)

		if appendEntriesResult.CurrentTerm > me.currentTerm {
			me.mux.Lock()
			me.currentTerm = appendEntriesResult.CurrentTerm
			me.leaderIndex = -1
			me.mux.Unlock()
		}
	}
}

// when calling this method use the go func format? -- this would launch it on a separate thread -- not sure how this would be different from the normal operation though (can experiment)
func sendLeaderHeartbeats() error {
	// if current server is the leader, then send AppendEntries heartbeat RPC at idle times to prevent leader election and just after election
	for {
		if me.serverIndex == me.leaderIndex {
			me.mux.Lock()
			allServers := serversUnion(me.newServers, me.activeServers)
			me.mux.Unlock()
			for _, i := range allServers {
				if i != me.serverIndex {
					go sendHeartbeat(i)
				}
			}
		} else {
			return nil
		}
		time.Sleep(time.Duration(me.leaderHeartBeatDuration) * time.Millisecond)
	}
}

func strToIntArr(strarr []string) []int {
	var ret []int
	for _, i := range strarr {
		index, _ := strconv.Atoi(i)
		ret = append(ret, index)
	}
	return ret
}

func applyEntries() {
	if me.lastAppliedIndex < me.commitIndex {
		me.lastAppliedIndex++
		logs := me.logs
		currentLog := logs[me.lastAppliedIndex]
		key := currentLog.Key
		value := currentLog.Value

		if !(key == "config-old-new" || key == "config-new") {
			// Add/update key value pair to the server if it is only a put request
			if value != "" {
				keyFound := false
				for i := 0; i < len(me.data); i++ {
					data := me.data[i]
					if data.Key == currentLog.Key {
						data.Value = currentLog.Value
						me.data[i] = data
						keyFound = true
						break
					}
				}
				if !keyFound {
					me.data = append(me.data, KeyValuePair{Key: currentLog.Key, Value: currentLog.Value})
				}

				data := me.data

				err := writeDataInFile(data)
				if err != nil {
					fmt.Println("error in writing data at file", err)
				}
			}
		}
	}
}

func applyCommittedEntries() {
	// at an interval of t ms, if (commitIndex > lastAppliedIndex), then apply entries one by one to the server file (state machine)
	for {
		if me.lastAppliedIndex < me.commitIndex {
			me.mux.Lock()
			applyEntries()
			me.mux.Unlock()
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

	for {

		var lastNap = rand.Intn(me.electionMaxTime-me.electionMinTime) + me.electionMinTime
		var asleep = time.Now().UnixNano()
		time.Sleep(time.Duration(lastNap) * time.Nanosecond)

		me.mux.Lock()
		activeServers := me.activeServers
		newServers := me.newServers
		me.mux.Unlock()

		allServers := serversUnion(activeServers, newServers)

		// no election necessary if the last message was received after we went to sleep.
		if me.lastMessageTime > asleep || me.serverIndex == me.leaderIndex || !isElementPresentInArray(allServers, me.serverIndex) {
			continue
		}

		/*
		 * oy, it's election time!
		 */
		// var electionTimeout = (time.Now().UnixNano()) + int64(rand.Intn(me.electionMaxTime-me.electionMinTime)+me.electionMinTime)
		// For leaderElection: Increment currentTerm
		me.mux.Lock()
		me.currentTerm++
		me.serverVotedFor = me.serverIndex
		currentElectionTerm := me.currentTerm
		lastLogEntryTerm := me.lastLogEntryTerm
		lastLogEntryIndex := me.lastLogEntryIndex
		myserverIndex := me.serverIndex
		me.mux.Unlock()

		metadataContent := [2]string{strconv.Itoa(currentElectionTerm), strconv.Itoa(myserverIndex)}
		err := ioutil.WriteFile(config.Servers[myserverIndex].Metadata, []byte(strings.Join(metadataContent[:], "\n")), 0)
		if err != nil {
			fmt.Println("inside leader election - Unable to write the new metadata information", err)
		}

		// Make RequestVote RPC calls to all other servers. (count its own vote +1) -- Do this async
		// forget the responses we've received so far.
		var requestVoteResponses = make(map[int]*RequestVoteResponse)

		type LE struct {
			mux                sync.Mutex
			majorityCounter    int
			majorityCounterNew int
			wg                 sync.WaitGroup
		}
		var le LE
		le.wg.Add(2)
		le.majorityCounter = int(math.Ceil(float64(len(activeServers)+1)/2)) - 1
		if len(newServers) == 0 {
			le.majorityCounterNew = 0
			le.wg.Done()
		} else {
			le.majorityCounterNew = int(math.Ceil(float64(len(newServers)+1) / 2))
			if isElementPresentInArray(newServers, me.serverIndex) {
				le.majorityCounterNew = le.majorityCounterNew - 1
			}
		}

		// request a vote from every client
		for _, index := range allServers {
			requestVoteResponses[index] = new(RequestVoteResponse)

			if index != myserverIndex {
				go func(index int) {
					// client, err := rpc.DialHTTP("tcp", config[index]["host"]+":"+config[index]["port"])
					conn, err := net.DialTimeout("tcp", config.Servers[index].Host+":"+config.Servers[index].Port, 200*time.Millisecond)

					if err != nil {
						fmt.Println("Candidate unable to create connection with another server ", err)
					} else {
						client := rpc.NewClient(conn)
						conn.SetDeadline(time.Now().Add(200 * time.Millisecond))
						defer client.Close()
						defer conn.Close()

						var voteRequest = RequestVoteArgs{
							CandidateIndex: myserverIndex,
							CandidateTerm:  currentElectionTerm,
							LastLogIndex:   lastLogEntryIndex,
							LastLogTerm:    lastLogEntryTerm,
						}
						err := client.Call("RaftServer.RequestVote", voteRequest, requestVoteResponses[index])

						if err != nil {
							fmt.Println("Error in request vote", err)
						}
						if requestVoteResponses[index].VoteGranted {
							le.mux.Lock()
							if isElementPresentInArray(activeServers, index) {
								le.majorityCounter--
								if le.majorityCounter == 0 {
									le.wg.Done()
								}
							}
							if isElementPresentInArray(newServers, index) {
								le.majorityCounterNew--
								if le.majorityCounterNew == 0 {
									le.wg.Done()
								}
							}
							le.mux.Unlock()
						} else {
							if requestVoteResponses[index].CurrentTerm > currentElectionTerm {
								me.mux.Lock()
								changedTerm := false
								if me.currentTerm < requestVoteResponses[index].CurrentTerm {
									changedTerm = true
									me.currentTerm = requestVoteResponses[index].CurrentTerm
									me.leaderIndex = -1
									metadataContent = [2]string{strconv.Itoa(me.currentTerm), "-1"}
								}
								me.mux.Unlock()
								le.mux.Lock()
								if le.majorityCounter > 0 {
									le.majorityCounter = -1
									le.wg.Done()
								}
								if le.majorityCounter > 0 || le.majorityCounterNew > 0 {
									le.majorityCounterNew = -1
									le.wg.Done()
								}
								le.mux.Unlock()

								if changedTerm {
									err := ioutil.WriteFile(config.Servers[myserverIndex].Metadata, []byte(strings.Join(metadataContent[:], "\n")), 0)
									if err != nil {
										fmt.Println("Unable to write the new metadata information", err)
									}
								}
							}
						}
					}
				}(index)
			}
		}
		timeout := 150 * time.Millisecond
		if !waitTimeout(&le.wg, timeout) {
			if currentElectionTerm == me.currentTerm {
				me.mux.Lock()
				me.leaderIndex = me.serverIndex
				fmt.Println("Leader elected")
				length := len(me.logs)
				for _, i := range allServers {
					me.nextIndex[i] = length
					me.matchIndex[i] = 0
				}
				me.mux.Unlock()
				go sendLeaderHeartbeats()
			}
		}
	}
}

// handling timeout for waitgroup
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

func isElementPresentInArray(array []int, ele int) bool {
	for _, i := range array {
		if i == ele {
			return true
		}
	}
	return false
}

func serversUnion(a []int, b []int) []int {
	ret := a
	for _, val := range b {
		if !isElementPresentInArray(ret, val) {
			ret = append(ret, val)
		}
	}
	return ret
}

// Called from PutKey, Also during heartbeat from leader every leaderHeartBeatDuration
// 2. Send AppendEntries to all other servers in async
// 3. When majority response received, add to the server file
// 4. If AppendEntries fail, need to update the lastMatchedLogIndex for that server and retry. this keeps retrying unless successful
// Also update the lastLogEntryIndex, commitIndex updated in appendentries
// Doubt: For log replication, we need the logs to follow the same term and index as leader, so changing the entries type to be LogEntry instead of KeyValuePair
// I think, logEntries will depend on next index index of the server, so need to send any logentries in this function as parameter.
func LogReplication(lastLogEntryIndex int) error {
	// filePath := config[me.serverIndex]["logfile"]
	type LR struct {
		mux                sync.Mutex
		majorityCounter    int
		majorityCounterNew int
		wg                 sync.WaitGroup
	}
	me.mux.Lock()
	activeServers := me.activeServers
	newServers := me.newServers
	me.mux.Unlock()

	allServers := serversUnion(activeServers, newServers)

	var lr LR
	lr.wg.Add(2)
	lr.majorityCounter = int(math.Ceil(float64(len(activeServers)+1)/2)) - 1 // leader counting it's own vote
	if len(me.newServers) == 0 {
		lr.majorityCounterNew = 0
		lr.wg.Done()
	} else {
		lr.majorityCounterNew = int(math.Ceil(float64(len(newServers)+1) / 2))
		if isElementPresentInArray(newServers, me.serverIndex) {
			lr.majorityCounterNew = lr.majorityCounterNew - 1
		}
	}

	var consensusErr error
	for _, index := range allServers {
		if index != me.serverIndex {
			go func(server int) {
				for {
					var logEntries []LogEntry

					// prepare logs to send
					me.mux.Lock()
					for j := me.nextIndex[server]; j < int(math.Min(float64(me.nextIndex[server]+1000), float64(len(me.logs)))); j++ {
						logEntries = append(logEntries, me.logs[j])
					}

					prevlogIndex := me.nextIndex[server] - 1
					prevlogTerm := "0"
					if prevlogIndex != -1 {
						prevlogTerm = me.logs[prevlogIndex].TermID
					}
					var leaderCurrentTerm = me.currentTerm
					var leaderIndex = me.leaderIndex
					var leaderCommitIndex = me.commitIndex
					me.mux.Unlock()

					prevlogTermInt, _ := strconv.Atoi(prevlogTerm)
					appendEntriesArgs := &AppendEntriesArgs{
						LeaderTerm:        leaderCurrentTerm,
						LeaderID:          leaderIndex,
						PrevLogIndex:      prevlogIndex,
						PrevLogTerm:       prevlogTermInt,
						Entries:           logEntries,
						LeaderCommitIndex: leaderCommitIndex,
					}

					var appendEntriesReturn AppendEntriesReturn

					conn, err := net.DialTimeout("tcp", config.Servers[server].Host+":"+config.Servers[server].Port, 200*time.Millisecond)
					if err != nil {
						fmt.Println("Leader unable to make connection for log replication", err)
						break
					} else {
						conn.SetDeadline(time.Now().Add(200 * time.Millisecond))
						client := rpc.NewClient(conn)
						defer client.Close()
						defer conn.Close()
						err := client.Call("RaftServer.AppendEntries", appendEntriesArgs, &appendEntriesReturn)
						if err == nil {
							if appendEntriesReturn.Success {
								me.mux.Lock()
								me.matchIndex[server] = prevlogIndex + len(logEntries)
								me.nextIndex[server] = me.matchIndex[server] + 1
								thisServerNextIndex := me.nextIndex[server]
								me.mux.Unlock()

								if thisServerNextIndex >= lastLogEntryIndex {
									lr.mux.Lock()
									if isElementPresentInArray(activeServers, server) {
										lr.majorityCounter--
										if lr.majorityCounter == 0 {
											lr.wg.Done()
										}
									}
									if isElementPresentInArray(newServers, server) {
										lr.majorityCounterNew--
										if lr.majorityCounterNew == 0 {
											lr.wg.Done()
										}
									}
									lr.mux.Unlock()
									break
								}
							} else {
								if appendEntriesReturn.CurrentTerm > leaderCurrentTerm {
									me.mux.Lock()
									me.leaderIndex = -1 // if current term of the server is greater than leader term, the current leader will become the follower.
									// Doubt: though this might not save the actual leader index.
									me.currentTerm = appendEntriesReturn.CurrentTerm
									metadataContent := [2]string{strconv.Itoa(me.currentTerm), "-1"}
									me.mux.Unlock()

									consensusErr = errors.New("Leader has changed state to follower, so consensus cannot be reached")
									lr.mux.Lock()
									if lr.majorityCounter > 0 {
										lr.wg.Done()
										lr.majorityCounter = -1
									}
									if lr.majorityCounterNew > 0 {
										lr.wg.Done()
										lr.majorityCounterNew = -1
									}
									lr.mux.Unlock()

									err := ioutil.WriteFile(config.Servers[me.serverIndex].Metadata, []byte(strings.Join(metadataContent[:], "\n")), 0)
									if err != nil {
										fmt.Println("Unable to write the new metadata information", err)
									}
									break
								}
								// next heartbeat request will send the new log entries starting from nextIndex[server] - 1
								me.mux.Lock()
								me.nextIndex[server] = me.nextIndex[server] - 1
								me.mux.Unlock()
							}
						} else {
							fmt.Println("error from append entry", err)
						}
					}
				}
			}(index)
		}
	}
	timeout := 500 * time.Millisecond
	if waitTimeout(&lr.wg, timeout) {
		return errors.New("Log Replication Timed out")
	}
	return consensusErr
}

//Init ... takes in config and index of the current server in config
func Init(index int) error {
	err := rpc.Register(&me)

	me.serverIndex = index
	me.leaderIndex = -1

	// TODO: commitIndex and lastAppliedIndex can also be saved in metadata file
	me.lastAppliedIndex = -1
	me.commitIndex = -1

	me.nextIndex = make(map[int]int)
	me.matchIndex = make(map[int]int)

	me.leaderHeartBeatDuration = 150
	me.electionMaxTime = 500000000
	me.electionMinTime = 350000000

	fileContent, _ := ioutil.ReadFile(config.Servers[me.serverIndex].LogFile)
	lines := []string{}
	if len(string(fileContent)) != 0 {
		lines = strings.Split(string(fileContent), "\n")
	}

	for index := range lines {
		line := strings.Split(lines[index], ",")
		thisLog := LogEntry{Key: line[0], Value: line[1], TermID: line[2], IndexID: line[3]}
		me.logs = append(me.logs, thisLog)
	}

	// loading data into memory
	fileContent, _ = ioutil.ReadFile(config.Servers[me.serverIndex].Filename)
	data := []string{}
	if len(string(fileContent)) != 0 {
		data = strings.Split(string(fileContent), "\n")
	}
	for index := range data {
		line := strings.Split(data[index], ",")
		thisdata := KeyValuePair{Key: line[0], Value: line[1]}
		me.data = append(me.data, thisdata)
	}

	me.lastLogEntryIndex = len(lines) - 1
	if me.lastLogEntryIndex < 0 {
		me.lastLogEntryTerm = 0
	} else {
		me.lastLogEntryTerm, _ = strconv.Atoi(strings.Split(lines[me.lastLogEntryIndex], ",")[2])
	}

	metadataContent, _ := ioutil.ReadFile(config.Servers[me.serverIndex].Metadata)

	if len(string(metadataContent)) != 0 {
		lines = strings.Split(string(metadataContent), "\n")
		me.currentTerm, _ = strconv.Atoi(lines[0])
		me.serverVotedFor, _ = strconv.Atoi(lines[1])
	} else {
		me.currentTerm = 0
		me.serverVotedFor = -1
	}

	go LeaderElection()
	go applyCommittedEntries() // running this over a separate thread for indefinite time

	if err != nil {
		fmt.Println("Format of service Task isn't correct. ", err)
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+config.Servers[index].Port)

	// Register a HTTP handler
	// rpc.HandleHTTP()

	// Listen to TPC connections on port 1234
	// listener, e := net.Listen("tcp", config[index]["host"]+":"+config[index]["port"])
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Listen error: ", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}

func runServer(serverIndex int) (RaftServer, error) {
	pid := os.Getpid()
	fmt.Printf("Server %d starts with process id: %d\n", serverIndex, pid)

	filename := config.Servers[serverIndex].Filename
	logFileName := config.Servers[serverIndex].LogFile
	metadataFileName := config.Servers[serverIndex].Metadata

	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		file, err := os.OpenFile(filename, os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create file ", err)
		}
		file.Close()
	}

	_, err = os.Stat(logFileName)
	if os.IsNotExist(err) {
		file, err := os.OpenFile(logFileName, os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create log file ", err)
		}
		file.Close()
	}

	_, err = os.Stat(metadataFileName)
	if os.IsNotExist(err) {
		file, err := os.OpenFile(metadataFileName, os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create meta data file ", err)
		}
		file.Close()
	}

	err = Init(serverIndex)

	return me, err
}

func readConfigFile() {
	configFile, err := os.Open("./config.json")
	if err != nil {
		fmt.Println("unable to read config file", err)
	}
	defer configFile.Close()
	configValue, _ := ioutil.ReadAll(configFile)
	json.Unmarshal(configValue, &config)
}

func readActiveServers() {
	fileContent, err := ioutil.ReadFile(activeServerFilename)
	if err != nil {
		fmt.Println("Unable to read active server file", err)
	}

	fileContentLines := strings.Split(string(fileContent), "\n")
	activeServersArr := strings.Split(fileContentLines[0], ",")
	for _, server := range activeServersArr {
		serverIndex, _ := strconv.Atoi(server)
		me.activeServers = append(me.activeServers, serverIndex)
	}

}

func main() {
	args := os.Args[1:]
	serverIndex, _ := strconv.Atoi(args[0])
	pid := os.Getpid()
	processFile := strconv.Itoa(serverIndex) + "-pid.txt"
	file, err := os.OpenFile(processFile, os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("Unable to create process file")
	}
	file.Close()
	_ = ioutil.WriteFile(processFile, []byte(strconv.Itoa(pid)), 0)
	readConfigFile()
	readActiveServers()
	runServer(serverIndex)
}
