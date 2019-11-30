% Reimplementing Raft

# Abstract

# Introduction

# Motivation

# Experimental Design

# Testing

## Integration with the DSLabs Distributed Service Test Framework

[DSLabs](https://github.com/emichael/dslabs) is a five-part assignment series containing both

While integration with the DSLabs test framework would have been possible for our project, it would not have provided a positive return on investment for this project.  DSLabs requires code to be written in Java, which would have required using [gobind](https://godoc.org/golang.org/x/mobile/cmd/gobind) or [cgo](https://golang.org/cmd/cgo/) to define a Go-to-Java or Go-to-C-to-Java interface, respectively.  That work would have required some effort to redesign the Raft server to run in multiple separate processes and receive commands via the command-line.  Unfortunately, that work would not have paid off in this case because the test suite was written for Paxos servers, assuming that data were exchanged according to Paxos semantics.  Since our server is a Raft server, with different synchronization assumptions, the tests would have provided false failures.  Alternatively, if a Raft test suite were created with the DSLabs framework, it would be very useful for future projects.

# Evaluation

# Related Works

# Conclusion
