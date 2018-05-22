#!/bin/bash

HOST=$1

if [ "$HOST" = '' ]; then
    echo './taillog.sh 172.16.23.1'
    exit
fi

ssh -t $HOST "bash -c 'ls ~/cutevpn/*.log | sort | tail -n1 | xargs tail -n 100 -f'"

