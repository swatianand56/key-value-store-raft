package main

import "sync"

type RaftServer struct {
	// who am I?
	ServerIndex int // my ID
	Logs        []LogEntry
	Data        []KeyValuePair

	// when is it?
	CurrentTerm       int // start value could be 0 (init from persistent storage during start)
	ServerVotedFor    int
	LastMessageTime   int64 // time in nanoseconds
	CommitIndex       int   // DOUBT: index of the last committed entry (should these 2 be added to persistent storage as well? How are we going to tell till where is the entry applied in the log?)
	LastAppliedIndex  int
	LastLogEntryIndex int
	LastLogEntryTerm  int

	// elections
	LeaderIndex             int // the current leader
	ElectionMinTime         int // time in nanoseconds
	ElectionMaxTime         int // time in nanoseconds
	LeaderHeartBeatDuration int

	// Log replication -- structures needed at leader
	// would be init in leader election
	NextIndex  map[int]int // this is to define which values to send for replication to each server during AppendEntries (initialize to last log index on leader + 1)
	MatchIndex map[int]int // not sure where this is used -- but keeping it for now (initialize to 0)

	Mux sync.Mutex

	ChangeMembership bool

	ActiveServers []int
	NewServers    []int
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
