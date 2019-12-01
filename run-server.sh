#! /usr/bin/env bash

build() {
    # build for clients and servers.
    pushd client
    go build
    popd

    pushd server
    go build
    popd
}

cleanServers() {
    # remove old state from server directory
    if [ -f serverprocs ]
    then
        cat serverprocs | xargs kill
    fi

    rm serverprocs serverlive
}

waitHere() {
    # Kill the server after we allow it.
    echo "Save a new value in serverlive to kill the server."
    while [ "`cat serverlive`" == "1" ]
    do
        sleep 2;
    done

    cleanServers
    ps
}

main() {
    # build servers
    build

    # start servers.
    pushd server

    # remove old state
    cleanServers
    echo "1" > serverlive

    # start servers and save directory.
    for i in {0..2}
    do
        ./server $i &
        echo $! >> serverprocs
    done

    # pause.
    waitHere &
    popd
}

main
