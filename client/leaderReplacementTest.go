package main

/*
someone must be able to be elected leader in a reasonably short amount of time.
*/

import (
	"fmt"
	"time"
)

var clusterSize = 5

func main() {
	TestLeaderReplacementLoop()
}

// func TestLeaderReplacementLoop(t *testing.T) {
func TestLeaderReplacementLoop() {
	stutters := []int{ /*50, 150, 250,*/ 500, 750, 1000}
	for _, stutter := range stutters {
		fmt.Println("Starting", stutter, "ms test.", time.Now())
		for i := 0; i < 20; i++ {
			xTestLeaderReplacement(time.Duration(stutter) * time.Millisecond)
		}
		fmt.Println("Finished", stutter, "ms test.", time.Now())
	}
}

// func xTestLeaderReplacement(t *testing.T, stutter time.Duration) {
func xTestLeaderReplacement(stutter time.Duration) {
	flags := 1 //VERBOSE.HEARTBEATS + VERBOSE.LIVENESS + VERBOSE.STATE
	slept := 0 * time.Millisecond
	var state = RaftServerSnapshot{}
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
		for iters = 0; leader == newLeader && iters < 100; iters++ {
			// put the leader to sleep for X time, then wait that long.
			err = ServerCall("Sleep", leader, int(stutter.Seconds()*1000), &slept)
			if err != nil {
				fmt.Println(err)
			}
			if iters > 10 {
				debugMessage(flags, -1, VERBOSE.HEARTBEATS, fmt.Sprintf("%d: Slept leader %d.", iters, newLeader))
			}
			time.Sleep(stutter)
			if iters > 10 {
				debugMessage(flags, -1, VERBOSE.HEARTBEATS, fmt.Sprintf("%d: Test woke up.", iters))
			}

			// FIXME: somehow we don't notice leader change when node 2 started as the leader.

			// see if we've picked a new leader then.
			ServerCall("GetState", (leader+1)%clusterSize, 0, &state)
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
