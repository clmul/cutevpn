#!/bin/bash

start() {
    sleep 1
    for ((;;)); do
        sudo killall -s INT cute
        if [ $? != 0 ]; then
            break
        fi
    done
    LOG=`date -u +%Y-%m-%d_%H.%M.%S`.log
    sudo ./cute >$LOG 2>&1 &
}

cd ~/cute
start >/dev/null 2>&1 &
