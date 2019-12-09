package main

/*
someone must be able to be elected leader in a reasonably short amount of time.
*/

import (
	"fmt"
	"testing"
)

func TestLeaderElection(t *testing.T) {
	flags := 0 // VERBOSE.HEARTBEATS + VERBOSE.LIVENESS + VERBOSE.STATE

	debugMessage(flags, -1, VERBOSE.STATE, "Starting leader election test.")
	leader, _, iters := ServerSetup(3, flags)

	if leader < 0 {
		debugMessage(flags, -1, VERBOSE.STATE, "Failed leader election test.")
		t.Errorf("Couldn't elect a leader in 10x 500ms rounds.")
	} else {
		debugMessage(flags, -1, VERBOSE.STATE, fmt.Sprintf("Finished leader election test in %d rounds", iters))
	}

	ServerTeardown(flags)
}
