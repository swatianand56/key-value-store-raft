package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	activeServerFilename = "./activeServers.cfg"
	activeServersArr     []string
	config               Config
)

type RaftServer struct {
	// who am I?
	serverIndex int // my ID
	logs        []LogEntry
	data        []KeyValuePair
	verbose     int
	awake       int64 // just a bool

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

	changeMembership bool

	activeServers []int
	newServers    []int
}

type RaftServerSnapshot struct {
	// a flat snapshot of the most recent data on a server
	// see server.go::GetState
	ActiveServers           string // actually stringified json-byte-array of int array: string(json.Marshal([]int))
	Awake                   int64
	ChangeMembership        bool
	CommitIndex             int
	CurrentTerm             int
	DataLength              int
	DataTail                KeyValuePair
	ElectionMaxTime         int
	ElectionMinTime         int
	LastAppliedIndex        int
	LastLogEntryIndex       int
	LastLogEntryTerm        int
	LastMessageTime         int64
	LeaderHeartBeatDuration int
	LeaderIndex             int
	LogLength               int
	LogTail                 LogEntry
	MatchIndex              map[int]int
	NewServers              []int
	NextIndex               map[int]int
	ServerIndex             int
	ServerVotedFor          int
	Verbose                 int
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

// verbose flags
type VerboseFlags struct {
	ALL         int // 1
	HEARTBEATS  int // 2
	CONNECTIONS int // 4
	READS       int // 8
	WRITES      int // 16
	VOTES       int // 32
	LIVENESS    int // 64
	STATE       int // 128
}

var VERBOSE VerboseFlags = VerboseFlags{1, 2, 4, 8, 16, 32, 64, 128}
var startTime = time.Now()

func debugMessage(flags int, serverIndex int, trigger int, message string) {
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
	case VERBOSE.VOTES:
		triggerName = "VOTE"
	case VERBOSE.LIVENESS:
		triggerName = "LIVENESS"
	case VERBOSE.STATE:
		triggerName = "STATE"
	default:
		triggerName = "ALL"
	}

	if flagOn(VERBOSE.ALL, flags) || flagOn(trigger, flags) {
		fmt.Printf("DEBUG | %s | %d | %.3fms | %s\n", triggerName, serverIndex, time.Since(startTime).Seconds()*1000, message)
	}
}

func flagOn(flag int, flags int) bool {
	return flags&flag == flag
}

func readConfigFile() {
	configFile, err := os.Open("./config.json")
	if err != nil {
		fmt.Println("unable to read config file", err)
	}
	defer configFile.Close()
	configValue, _ := ioutil.ReadAll(configFile)
	error := json.Unmarshal(configValue, &config)
	if error != nil {
		fmt.Println("unmarshall error:", err)
	}
}

func readActiveServers(me *RaftServer) {
	fileContent, err := ioutil.ReadFile(activeServerFilename)
	if err != nil {
		fmt.Println("Unable to read active server file", err)
	}

	fileContentLines := strings.Split(string(fileContent), "\n")
	activeServersArr = strings.Split(fileContentLines[0], ",")

	if me != nil {
		for _, server := range activeServersArr {
			serverIndex, _ := strconv.Atoi(server)
			me.activeServers = append(me.activeServers, serverIndex)
		}
		debugMessage(me.verbose, me.serverIndex, VERBOSE.LIVENESS, fmt.Sprintf("active servers: %v", me.activeServers))
	}
}
