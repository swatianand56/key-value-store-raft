package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"time"
)

var server0, server1, server2 *os.Process

var serverList = []string{
	"localhost:8001",
	"localhost:8002",
	"localhost:8003",
}

func ServerSleep(serverIndex int, sleepTime time.Duration, sleptTime *time.Duration) error {
	conn, err := net.DialTimeout("tcp", serverList[serverIndex], (sleepTime+250)*time.Millisecond)

	if err == nil {
		client := rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add((sleepTime + 250) * time.Millisecond))
		err = client.Call("RaftServer.Sleep", sleepTime, &sleptTime)
	}

	return err
}

func GetState(serverIndex int, serverState RaftServer) error {
	conn, err := net.DialTimeout("tcp", serverList[serverIndex], 250*time.Millisecond)

	if err == nil {
		client := rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(250 * time.Millisecond))
		err = client.Call("RaftServer.CurrentState", 0, &serverState)
	}

	return err
}

func ServerCall(serverCall string, serverIndex int, input interface{}, output interface{}) error {
	conn, err := net.DialTimeout("tcp", serverList[serverIndex], 250*time.Millisecond)

	if err == nil {
		client := rpc.NewClient(conn)
		defer client.Close()
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(250 * time.Millisecond))
		err = client.Call("RaftServer."+serverCall, input, output)
	}

	fmt.Println("ServerCall output:", output)

	return err
}

func ServerSetup() {
	attr1 := new(os.ProcAttr)
	attr1.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	attr2 := new(os.ProcAttr)
	attr2.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	attr3 := new(os.ProcAttr)
	attr3.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}

	server0, err = os.StartProcess("../server/server", []string{"../server/server", "0"}, attr1)
	if err != nil {
		fmt.Println("Server 0 Error:", err)
	}
	server1, err = os.StartProcess("../server/server", []string{"../server/server", "1"}, attr2)
	if err != nil {
		fmt.Println("Server 1 Error:", err)
	}
	server2, err = os.StartProcess("../server/server", []string{"../server/server", "2"}, attr3)
	if err != nil {
		fmt.Println("Server 2 Error:", err)
	}
}

func ServerTeardown() {
	if server0 != nil {
		server0.Kill()
	}
	if server1 != nil {
		server1.Kill()
	}
	if server2 != nil {
		server2.Kill()
	}
}