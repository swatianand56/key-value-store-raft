# Notes

## References

### Teaching Rigorous Distributed Systems with Efficient Model Checking

Mike would like us to investigate whether TRDSwEMC is pedagogically useful.

https://homes.cs.washington.edu/~mernst/pubs/dslabs-eurosys2019.pdf

### Distributed Systems Labs and Framework

The companion repo to the paper.

https://github.com/emichael/dslabs

### Alternate Raft Implementations

https://raft.github.io/

| Repository                         | License        | Election | Membership | Compaction | Updated    |
| ---------------------------------- | -------------- | -------- | ---------- | ---------- | ---------- |
| github.com/lni/dragonboat          | Apache2        | Yes      | Yes        | Yes        | 2019-02-10 |
| github.com/dev-urandom/graft       |                | Partial  |            |            | 2013-10-24 |
| github.com/goraft/raft             | MIT            | Yes      | Partial?   | Yes        | 2013-07-05 |
| github.com/coreos/etcd             | Apache 2.0     | Yes      | Yes        | Yes        | 2014-10-27 |
| github.com/hashicorp/raft          | MPL-2.0        | Yes      | Yes        | Yes        | 2014-04-21 |
| bitbucket.org/jpathy/raft          | WTFPL          |          |            |            | 2014-07-24 |
| github.com/peterbourgon/raft       | Simplified BSD | Yes      | Yes        | No         | 2013-07-05 |
| github.com/mreiferson/pontoon      |                |          |            |            | 2013-09-02 |
| github.com/lionelbarrow/seaturtles |                |          |            |            | 2013-09-02 |

# key-value-store-raft

Key Value Store Implementation Using Raft Consensus Algorithm

We are planning to implement a distributed key-value store using the Raft consensus algorithm. We are
building the cluster of 5 servers that can tolerate up to 2 server failures. Servers will communicate with
each other using the RPC calls so we will be implementing the two types of RPCs, RequestVote RPC and
AppendEntries RPC. RequestVote RPCs would be initiated by the candidate to contest for the leader
election, whereas AppendEntries RPCs would be sent by the leader to replicate the logs to the other
servers and would also be used as the heartbeat. Our algorithm would contain the following modules:

1. <b>Leader Election</b>: All servers initially start in the follower state and would be able to initiate the
leader election for a new term when the random timeouts expire. The server would broadcast the
RequestVote RPC to other servers and wait for majority votes. The server would retry with a
higher term if it does not win with majority votes. If in the meantime, it receives AppendEntries
RPC from another server for the same or higher-numbered term, it steps down to the follower
state.

2. <b>Safety</b>: In the RequestVote RPC, if the server’s current term is higher than the candidate’s term or
if they both have the same term and the log index of the current server is higher than the
candidate’s log index, then the server will reject its vote. Whenever a new leader is elected, it’ll
run the AppendEntries consistency check on the followers and match the follower logs with the
leader log.

3. <b>Log Replication</b>: The leader will accept the key-value from the client and sends the
AppendEntries RPC to all the follower servers and wait for the acknowledgment of the majority
of the servers before committing. This will ensure that the log has been replicated at the majority
of the servers.

# Evaluation:
We will be testing our algorithm on 5 replicas that’ll be running on VM’s hosted on a public
cloud. We would be using our Key-Value store implemented for Project 1 as the baseline and compare its
performance and correctness with the current consensus-based implementation.

<b>Correctness and Failure Testing</b>: We would run a normal case where all servers are up and running
smoothly, we will verify at regular intervals that all the inserted keys should be present at majority of
servers. One final test case would be that all the keys should be present at all the servers after a small
time-window. We would also test the system for failure and straggler behavior of followers and leader
and a combination of both up till the maximum allowed server failures (n/2 - 1) for the Raft consensus
algorithm to work perfectly. For the stragglers, we will also test with the majority of slow servers and
check how the system behaves in such a case.

<b>Performance Testing</b>: We would use different ratios of read-write workloads and compare against our
earlier version of the key-value store.

The experiments would also include:
1. The average time taken to elect the new leader by varying the election timeouts (0-5 ms, 12-25 ms, 150-300 ms, 300-500 ms)
2. Measure the throughput/latency by varying the number of nodes in the cluster (3,5,7,9,11)

# TODOs

List of TODOs:

    egrep -nHr "//.*[A-Z]{4}" */*.go
