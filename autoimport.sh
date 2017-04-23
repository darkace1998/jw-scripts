#!/bin/bash
#
# autoimport.sh
#

if [[ $1 = --help ]]; then
    cat<<EOF
Usage: autoimport.sh USER DIRECTORY
Import videos to jwb-rpi.

Usually executed by devmon. Must be run as root.
EOF
    exit
fi

user="$1"
dir="$2"

[[ $user && $dir ]] || exit 1

systemctl is-active --quiet jwb-rpi@$user && player=1
systemctl is-active --quiet jwb-rpi-downloader@$user && downloader=1
systemctl stop jwb-rpi@$user jwb-rpi-downloader@$user || exit 1

su -c 'omxplayer --loop /usr/share/jw-scripts/spinner' $user &>/dev/null & spinner=$!

su -c "/usr/local/bin/jwb-rpi-downloader --import '$dir'" $user || echo "jwb-rpi-downloader exited with status $?" 1>&2

kill $spinner

((player)) && systemctl start jwb-rpi@$user
((downloader)) && systemctl start jwb-rpi-downloader@$user