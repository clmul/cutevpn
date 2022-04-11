#!/bin/bash

search=$1
if [ "$search" = '' ]; then
    sudo iptables -t mangle -L -v | grep 'MARK set 0x7e4' | awk '{print $11}'
    exit
fi

for path in $(find /sys/fs/cgroup -type d -name "*$search*.scope" -printf '%P\n'); do
    echo -n "bypass $path (Y/n)?"
    read answer
    if [ "$answer" = 'n' ]; then
        continue
    fi
    cmd="iptables -t mangle -A OUTPUT -m cgroup --path $path -j MARK --set-mark 2020"
    echo $cmd
    sudo $cmd
done
