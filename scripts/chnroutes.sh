#!/bin/bash

curl https://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest | awk -F '|' '/CN\|ipv4/ {print $4 "/" log($5)/log(2)}'
