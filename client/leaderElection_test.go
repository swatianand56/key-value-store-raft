package main

import (
	"fmt"
	"testing"
	"time"
)

func TestLeaderElection(t *testing.T) {
	var slept *time.Duration
	currState := RaftServer{}

	ServerSetup()

	// let servers start up.
	time.Sleep(500 * time.Millisecond)

	// put server 0 to sleep
	err = ServerSleep(0, 1000, slept)

	if err == nil {
		// let peers react to loss of leader.
		time.Sleep(1000 * time.Millisecond)

		// get current node state.
		// err = ServerCall("CurrentState", 1, 0, &currState)
		err = GetState(1, currState)

		fmt.Println("TestLeaderElection", currState)

		if err == nil {
			if currState.LeaderIndex == 0 {
				t.Errorf("After sleeping for 500 ms, server 0 is still unexpectedly the leader!")
			}
			if currState.LeaderIndex == -1 {
				t.Errorf("After sleeping for 500 ms, no server is the leader!")
			}
		} else {
			t.Errorf("Could not get server state: %s", err)
		}
	} else {
		t.Errorf("Slept for %d ms: %s", slept, err)
	}

	ServerTeardown()
}
