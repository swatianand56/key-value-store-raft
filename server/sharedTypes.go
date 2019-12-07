package main

import (
	"fmt"
	"sync"
	"time"
)

const numServers = 3

type RaftServer struct {
	// who am I?
	serverIndex int // my ID
	logs        []LogEntry
	data        []KeyValuePair
	verbose     int
	alive       bool

	// when is it?
	currentTerm       int // start value could be 0 (init from persistent storage during start)
	serverVotedFor    int
	lastMessageTime   int64 // time in nanoseconds
	commitIndex       int   // DOUBT: index of the last committed entry (should these 2 be added to persistent storage as well? How are we going to tell till where is the entry applied in the log?)
	lastAppliedIndex  int
	lastLogEntryIndex int
	lastLogEntryTerm  int

	// elections
	leaderIndex             int // the current leader
	electionMinTime         int // time in nanoseconds
	electionMaxTime         int // time in nanoseconds
	leaderHeartBeatDuration int

	// Log replication -- structures needed at leader
	// would be init in leader election
	nextIndex  [numServers]int // this is to define which values to send for replication to each server during AppendEntries (initialize to last log index on leader + 1)
	matchIndex [numServers]int // not sure where this is used -- but keeping it for now (initialize to 0)

	mux sync.Mutex
}

type RaftServerSnapshot struct {
	// a flat snapshot of the most recent data on a server
	// see server.go::GetState
	CommitIndex int
	CurrentTerm int
	DataLength  int
	// DataTail                KeyValuePair
	ElectionMaxTime         int
	ElectionMinTime         int
	LastAppliedIndex        int
	LastLogEntryIndex       int
	LastLogEntryTerm        int
	LastMessageTime         int64
	LeaderHeartBeatDuration int
	LeaderIndex             int
	LogLength               int
	// LogTail                 LogEntry
	// MatchIndex              [numServers]int
	// NextIndex               [numServers]int
	ServerIndex    int
	ServerVotedFor int
	Verbose        int
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
	CandidateIndex int
	CandidateTerm  int
	LastLogIndex   int
	LastLogTerm    int
}

type RequestVoteResponse struct {
	CurrentTerm int
	VoteGranted bool
}

// verbose flags
type VerboseFlags struct {
	ALL         int // 1
	HEARTBEATS  int // 2
	CONNECTIONS int // 4
	READS       int // 8
	WRITES      int // 16
}

var VERBOSE VerboseFlags = VerboseFlags{1, 2, 4, 8, 16}

func debugMessage(server *RaftServer, trigger int, message string) {
	//
	//	server: server ID: me.serverIndex
	//	trigger: verbose flag that triggered the debug: HEARTBEAT, CONNECTION
	//	message: thing to print: "Sent Heartbeat!"
	//
	var triggerName string

	switch trigger {
	case VERBOSE.HEARTBEATS:
		triggerName = "HEARTBEAT"
	case VERBOSE.CONNECTIONS:
		triggerName = "CONNECTION"
	case VERBOSE.READS:
		triggerName = "READ"
	case VERBOSE.WRITES:
		triggerName = "WRITE"
	default:
		triggerName = "ALL"
	}

	if flagOn(VERBOSE.ALL, server.verbose) || flagOn(trigger, server.verbose) {
		fmt.Printf("%s | %d | %d | %s\n", triggerName, server.serverIndex, time.Now().UnixNano()/int64(time.Millisecond)%1000000, message)
	}
}

func flagOn(flag int, flags int) bool {
	return flags&flag == flag
}
