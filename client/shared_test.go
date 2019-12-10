package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var servers []*os.Process

// make an arbitrary call to the specified server.
func ServerCall(serverCall string, serverIndex int, input interface{}, output interface{}) error {
	return ServerCallTime(serverCall, serverIndex, input, output, 250*time.Millisecond)
}

// make an arbitrary call to the specified server with a 30-second timeout.
func ServerCallLong(serverCall string, serverIndex int, input interface{}, output interface{}) error {
	return ServerCallTime(serverCall, serverIndex, input, output, 30*time.Second)
}

// make an arbitrary call to the specified server with a custom timeout.
func ServerCallTime(serverCall string, serverIndex int, input interface{}, output interface{}, deadline time.Duration) error {
	conn, err := net.DialTimeout("tcp", config.Servers[serverIndex].Host+":"+config.Servers[serverIndex].Port, 250*time.Millisecond)

	if err == nil {
		client := rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(deadline))
		err = client.Call("RaftServer."+serverCall, input, output)
	}

	return err
}

// make an arbitrary call to the any server with a custom timeout.
// good for explicitly calling server 0 before the servers array has been initialized.
func ServerCallByIp(serverCall string, server string, input interface{}, output interface{}) error {
	conn, err := net.DialTimeout("tcp", server, 250*time.Millisecond)

	if err == nil {
		client := rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(250 * time.Millisecond))
		err = client.Call("RaftServer."+serverCall, input, output)
	}

	return err
}

/*
start up a cluster with the specified number of servers and debug flags.

1. starts up the specified number of servers.
2. runs the config manager just once to teach each live server about the others
3. tries to identify the leader by querying server0 (localhost:8001) every 500ms, up to 10 times.
4. returns the active server ID, the list of active servers, and the number of iterations required to identify a leader.
*/
func ServerSetup(numServers int, verboseFlags int) (int, []int, int) {
	// see sharedTypes.go::VerboseFlags:
	//
	// ALL          1
	// HEARTBEATS   2
	// CONNECTIONS  4
	// READS        8
	// WRITES       16
	// VOTES        32
	// LIVENESS     64
	// STATE        128
	//
	// returns: the leader server.

	leader := -1

	if numServers < 1 {
		fmt.Println("ServerSetup numServers")
	}

	var server *os.Process

	servers = make([]*os.Process, numServers)

	// start servers
	for i := 0; i < numServers; i++ {
		attr := new(os.ProcAttr)
		attr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
		server, err = os.StartProcess("../server/server", []string{"../server/server", strconv.Itoa(i), strconv.Itoa(verboseFlags)}, attr)
		if err != nil {
			fmt.Println("Server 0 Error:", err)
		} else {
			debugMessage(verboseFlags, i, VERBOSE.LIVENESS, "Starting server.")
			servers[i] = server
		}
	}

	// give them half a second to start up.
	time.Sleep(500 * time.Millisecond)

	// synchronize live servers
	cmd := exec.Command("../config_manager/config_manager", "--hurry", "--once")
	err := cmd.Run()

	// couldn't sync servers, there's nothing to test.
	if err != nil {
		fmt.Println("ServerSetup config_manager_policy err:", err)
		ServerTeardown(verboseFlags)
		return -1, []int{-1}, -1
	}

	var state *RaftServerSnapshot
	state = &RaftServerSnapshot{}
	// get current leader
	i := 0
	for i = 0; leader < 0 && i < 200; i++ {
		state = &RaftServerSnapshot{}
		err = ServerCallByIp("GetState", "localhost:8001", 0, state)

		if err == nil {
			leader = state.LeaderIndex
		} else {
			fmt.Printf("err: %s", err.Error())
			err = nil
		}

		if leader == -1 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	readConfigFile()

	var activeServers []int
	err = json.Unmarshal([]byte(state.ActiveServers), &activeServers)

	return leader, activeServers, i
}

func ServerTeardown(verboseFlags int) {
	for i, server := range servers {
		if server != nil {
			debugMessage(verboseFlags, i, VERBOSE.LIVENESS, "Killing server.")
			server.Kill()
		}
	}
}
