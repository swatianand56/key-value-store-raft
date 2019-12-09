package main

// Leader not part of new config — new leader should eventually get elected
// newly added servers should have all logs after cnew, but old servers should not have
// (all logs before that should match on all new and old servers)
// simultaneous client requests (10k keys put)
// (also put sleep() between coldnew and new)

//Task to do before running this test - Run 5 servers mentioned in the serverlist and run three parallel clients
// func TestMembershipChanges3(t *testing.T) {
// removeTextFile()

// 	path := "./../../server/"
// 	// var activeServerFilename = "./activeServers.cfg"
// 	// oldConfigStr := "0,1,2"
// 	serverToAddInNewConfigStr := "3,4" // and leader
// 	serverList := []string{
// 		"localhost:8001",
// 		"localhost:8002",
// 		"localhost:8003",
// 		"localhost:8004",
// 		"localhost:8005",
// 	}

// 	// servers := strings.Split(oldConfigStr, ",") // 2n+1 servers
// 	// newServersArr := strings.Split(serverToAddInNewConfigStr, ",")

// 	var number_of_keys = 50
// 	kv739_init(serverList, 11)
// 	var oldValue string
// 	var keys_not_found = 0
// 	var key_value_mismatch = 0
// 	var put_key_unsuccessful = 0

// 	for i := 0; i < number_of_keys; i++ {
// 		var key = strconv.Itoa(i)
// 		x := kv739_put(key, key, &oldValue)
// 		y := kv739_get(key, &oldValue)
// 		if x != -1 {
// 			if y == -1 {
// 				keys_not_found++
// 			}
// 			if key != oldValue {
// 				key_value_mismatch++
// 			}
// 		} else {
// 			put_key_unsuccessful++
// 		}
// 	}

// 	currentLeader := strconv.Itoa(leaderIndex)
// 	newServerConfig := serverToAddInNewConfigStr
// 	if currentLeader == "0" {
// 		newServerConfig = "1,2," + serverToAddInNewConfigStr
// 	} else if currentLeader == "1" {
// 		newServerConfig = "0,2," + serverToAddInNewConfigStr
// 	} else {
// 		newServerConfig = "0,1," + serverToAddInNewConfigStr
// 	}

// 	newServerConfigArr := strings.Split(newServerConfig, ",")

// 	var newConfigServers = []int{}
// 	for _, i := range newServerConfigArr {
// 		j, err := strconv.Atoi(i)
// 		if err != nil {
// 			panic(err)
// 		}
// 		newConfigServers = append(newConfigServers, j)
// 	}

// 	time.Sleep(time.Duration(1) * time.Second) // buffer time to sync logs int all the servers

// 	fmt.Println("new config servers -----------------------------------------> ", newConfigServers)
// 	fmt.Println("membership changes status ----------------------------------> ", kv739_changeMembership(newConfigServers))

// 	majority := int(math.Ceil(float64(len(newConfigServers)+1) / 2))
// 	for i := 0; i < number_of_keys; i++ {
// 		var key = strconv.Itoa(i)
// 		x := kv739_put(key, key, &oldValue)
// 		y := kv739_get(key, &oldValue)
// 		if x != -1 {
// 			if y == -1 {
// 				keys_not_found++
// 			}
// 			if key != oldValue {
// 				key_value_mismatch++
// 			}
// 		} else {
// 			put_key_unsuccessful++
// 		}
// 	}

// 	currentLeaderIndex := strconv.Itoa(leaderIndex)
// 	time.Sleep(time.Duration(1) * time.Second) // buffer time to sync logs int all the servers, probably should wait more as client might be still running

// 	arr := newServerConfigArr // check the logs of new server config
// 	var content = make(map[string][]string)
// 	leaderLogSize := 0
// 	for i := range arr {
// 		index := arr[i]
// 		filePath := path + "log-" + index + ".txt"
// 		fileContent, err := ioutil.ReadFile(filePath)
// 		if err != nil {
// 			t.Errorf("error in reading log file")
// 			ServerTeardown(arr)
// 			return
// 		}
// 		lines := strings.Split(string(fileContent), "\n")
// 		content[index] = lines
// 		if index == currentLeaderIndex {
// 			leaderLogSize = len(lines)
// 		}
// 	}

// 	count := 0
// 	for i := range arr {
// 		index := arr[i]
// 		if len(content[index]) == leaderLogSize {
// 			count++
// 		}
// 	}
// 	if count < majority {
// 		t.Errorf("Failed test case: majority of servers should have the same length of logs as leader")
// 	}

// 	for i := 0; i < leaderLogSize; i++ {
// 		count := 1
// 		line := content[strconv.Itoa(leaderIndex)][i]
// 		for j := range arr {
// 			index := arr[j]
// 			if index != currentLeaderIndex {
// 				if i < len(content[index]) && line == content[index][i] {
// 					count++
// 				}
// 			}
// 		}
// 		if count < majority {
// 			t.Errorf("Failed test case: majority of servers log content is not matching")
// 		}
// 	}

// 	// ServerTeardown(servers)       // kill old config servers
// 	// ServerTeardown(newServersArr) // kill newwly added servers
// }
