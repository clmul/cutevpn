#!/bin/bash

set -e

deploy() {
    conf=$1
    host=${conf%.*}
    echo deploy $host
    ssh $host 'mkdir -p ~/cutevpn'

    export GOOS=$(ssh $host 'bash -c "go env GOOS"')
    export GOARCH=$(ssh $host 'bash -c "go env GOARCH"')
    echo $GOOS $GOARCH

    go build github.com/clmul/cutevpn/cutevpn
    ssh $host 'rm ~/cutevpn/cutevpn' || true
    scp cutevpn $host:~/cutevpn/
    scp $conf $host:~/cutevpn/config.toml
    scp start.sh $host:~/cutevpn/

    ssh $host 'bash -c "~/cutevpn/start.sh"'
    rm cutevpn
}

for m in $@; do
    deploy "$m.conf"
done

if [ "$1" = '' ]; then
    for conf in $(ls *.conf); do
        deploy $conf
    done
fi
