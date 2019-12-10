package main

/*
someone must be able to be elected leader in a reasonably short amount of time.
*/

import (
	"fmt"
	"testing"
)

func TestLeaderElection(t *testing.T) {
	return

	flags := VERBOSE.STATE // VERBOSE.HEARTBEATS + VERBOSE.LIVENESS + VERBOSE.STATE

	debugMessage(flags, -1, VERBOSE.STATE, "Starting leader election test.")
	leader, _, iters := ServerSetup(clusterSize, flags)

	if leader < 0 {
		debugMessage(flags, -1, VERBOSE.STATE, fmt.Sprintf("Failed leader election test: %d", iters))
		t.Errorf(fmt.Sprintf("Couldn't elect a leader in %d rounds.", iters))
	} else {
		debugMessage(flags, -1, VERBOSE.STATE, fmt.Sprintf("Finished leader election test in %d rounds", iters))
	}

	ServerTeardown(flags)
}
