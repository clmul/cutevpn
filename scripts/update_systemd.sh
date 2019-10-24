#!/bin/bash

set -e

deploy() {
    conf=$1
    host=${conf%.*}
    echo update cutevpn on $host
    ssh $host 'mkdir -p ~/cute'

    export GOOS=$(ssh $host 'bash -c "go env GOOS"')
    export GOARCH=$(ssh $host 'bash -c "go env GOARCH"')
    echo $GOOS $GOARCH

    go build -o cutevpn github.com/clmul/cutevpn/cutevpn
    scp cutevpn $host:~
    scp ../cutevpn.service $host:~
    scp $conf $host:~/cutevpn.toml
    ssh $host 'cat > ~/update_cutevpn.sh' <<EOF
sudo rm /usr/local/bin/cutevpn
sudo mv ~/cutevpn /usr/local/bin/
sudo mv ~/cutevpn.toml /etc/
sudo mv ~/cutevpn.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl restart cutevpn
rm \$0
EOF
    ssh $host 'bash ~/update_cutevpn.sh'
    rm cutevpn
}

cd conf.d
for m in $@; do
    deploy "$m.conf"
done
