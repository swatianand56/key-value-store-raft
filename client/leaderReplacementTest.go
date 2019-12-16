package main

/*
someone must be able to be elected leader in a reasonably short amount of time.
*/

import (
	"fmt"
	"math/rand"
	"time"
)

var clusterSize = 9
var maxRounds = 20

func main() {
	TestLeaderReplacementLoop()
}

// func TestLeaderReplacementLoop(t *testing.T) {
func TestLeaderReplacementLoop() {
	stutters := []int{50, 150, 250, 500, 750, 1000}
	for _, stutter := range stutters {
		fmt.Println("Starting", stutter, "ms test.", time.Now())
		for i := 0; i < 100; i++ {
			xTestLeaderReplacement(time.Duration(stutter) * time.Millisecond)
		}
		fmt.Println("Finished", stutter, "ms test.", time.Now())
	}
}

// func xTestLeaderReplacement(t *testing.T, stutter time.Duration) {
func xTestLeaderReplacement(stutter time.Duration) {
	flags := VERBOSE.STATE //VERBOSE.HEARTBEATS + VERBOSE.LIVENESS + VERBOSE.STATE
	slept := 0 * time.Millisecond
	iters := 0

	debugMessage(flags, -1, VERBOSE.STATE, "Electing a leader for leader replacement test.")
	leader, _, _ := ServerSetup(clusterSize, flags)

	if leader < 0 {
		debugMessage(flags, -1, VERBOSE.STATE, "Failed to elect a leader for leader replacement test.")
		// t.Errorf("Couldn't elect a leader.")
	} else {

		debugMessage(flags, -1, VERBOSE.STATE, fmt.Sprintf("Replacing %s stuttering leader, %d.", stutter, leader))

		newLeader := leader
		// TODO: for full perf data // runTime := time.Now()
		// TODO: for full perf data // for iters = 0; leader == newLeader && time.Since(runTime) < 60*time.Second; iters++ {
		for iters = 0; iters < maxRounds && (newLeader == leader || newLeader == -1); iters++ {
			// put the leader to sleep for X time, then wait that long.
			err = ServerCall("Sleep", leader, int(stutter.Seconds()*1000), &slept)
			if err != nil {
				fmt.Println(err)
			}
			if iters > 10 {
				debugMessage(flags, -1, VERBOSE.HEARTBEATS, fmt.Sprintf("%d: Slept leader %d.", iters, leader))
			}
			time.Sleep(stutter)
			if iters > 10 {
				debugMessage(flags, -1, VERBOSE.HEARTBEATS, fmt.Sprintf("%d: Test woke up.", iters))
			}

			// FIXME: somehow we don't notice leader change to zero when node 2 started as the leader.
			queryServer := rand.Intn(clusterSize)

			// see if we've picked a new leader then.
			var state = RaftServerSnapshot{LeaderIndex: -1, ServerIndex: -1}
			ServerCall("GetState", queryServer, 0, &state)
			debugMessage(flags, -1, VERBOSE.HEARTBEATS, fmt.Sprintf("%d: Found state: %#+v.", iters, state))
			newLeader = state.LeaderIndex
		}

		state := "Finished"
		if leader == newLeader {
			state = "Failed"
		}

		debugMessage(flags, -1, VERBOSE.STATE, fmt.Sprintf("%s leader replacement test in %d rounds with %s stutter on %d node cluster.", state, iters, stutter, clusterSize))
	}

	ServerTeardown(flags)
}
