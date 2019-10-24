#!/bin/bash

set -e

deploy() {
    conf=$1
    host=${conf%.*}
    conf="conf.d/$conf"
    echo deploy $host
    ssh $host 'mkdir -p ~/cute'

    export GOOS=$(ssh $host 'bash -c "go env GOOS"')
    export GOARCH=$(ssh $host 'bash -c "go env GOARCH"')
    echo $GOOS $GOARCH

    go build -o cute github.com/clmul/cutevpn/cutevpn
    ssh $host 'rm ~/cute/cute' || true
    scp cute $host:~/cute/
    scp $conf $host:~/cute/config.toml
    scp start.sh $host:~/cute/

    ssh $host 'bash -c "~/cute/start.sh"'
    rm cute
}

for m in $@; do
    deploy "$m.conf"
done

if [ "$1" = '' ]; then
    for conf in $(ls *.conf); do
        deploy $conf
    done
fi
