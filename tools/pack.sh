#!/bin/bash

package() {
    dir="cutevpn_${TRAVIS_TAG}_${GOOS}_${GOARCH}"
    echo $dir
    mkdir "$dir"
    cp config.toml.example "$dir/config.toml"
    cd "$dir"
    go build "$gopackage"
    cd ..
    tar zcvf "$dir.tar.gz" "$dir"
}

gopackage="github.com/clmul/cutevpn/cutevpn"
cd "$GOPATH/src/$gopackage"

export GOOS=linux
export GOARCH=amd64
package

export GOOS=darwin
export GOARCH=amd64
package
