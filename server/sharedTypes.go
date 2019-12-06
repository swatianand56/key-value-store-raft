package main

import "sync"

type RaftServer struct {
	// who am I?
	serverIndex int // my ID
	logs        []LogEntry
	data        []KeyValuePair

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
	nextIndex  map[int]int // this is to define which values to send for replication to each server during AppendEntries (initialize to last log index on leader + 1)
	matchIndex map[int]int // not sure where this is used -- but keeping it for now (initialize to 0)

	mux sync.Mutex
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
