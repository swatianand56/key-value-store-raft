% Reimplementing Raft

# Abstract

# Introduction

# Motivation

# New Raft Features

## Dynamic Load Balancing

### Limitations

Currently, dynamic load balancing doesn't add servers in bulk.  Rather, once a single server requests to be added to the pool the work request is immediately executed by the leader, requiring one round per added server.  A future improvement could be to bulk-add new servers outside of the server-startup process.  This would allow the spontaneous merging of clusters.  However, since cluster merging was not a goal, this approach was not a necessary to implement at this time, and keeps the cluster addition process simpler.

# Experimental Design

# Testing

## Integration with the DSLabs Distributed Service Test Framework

[DSLabs](https://github.com/emichael/dslabs) is a five-part assignment series containing both single-threaded test scripts and a visual debugger.  The single-threaded nature of the test scripts allows the tests to execute deterministically, even in a distributed system.  The visual debugger allows users to step the system forward and backward through time while trying different branches in the state tree.

While integration with the DSLabs test framework would have been possible for our project, it would not have provided a positive return on investment for this project.  DSLabs requires code to be written in Java, which would have required using [gobind](https://godoc.org/golang.org/x/mobile/cmd/gobind) or [cgo](https://golang.org/cmd/cgo/) to define a Go-to-Java or Go-to-C-to-Java interface, respectively.  That work would have required some effort to redesign the Raft server to run in multiple separate processes and receive commands via the command-line.  Unfortunately, that work would not have paid off in this case because the test suite was written for Paxos servers, assuming that data were exchanged according to Paxos semantics.  Since our server is a Raft server, with different synchronization assumptions, the tests would have provided false failures.  Alternatively, if a Raft test suite were created with the DSLabs framework, it would be very useful for future projects.

We were able to reuse the concept behind DSLabs to create single-threaded test scripts that sequentially moved the system through different states and verify correct behavior during each transition.  Further, we were able to learn from [other Raft implementations](https://raft.github.io/) and reimplement concepts from their tests in our own test suite.

##

# Evaluation

# Future Direction

- Change leader election delay during repeated leader failures to handle correlated failures.

# Related Works

# Conclusion
