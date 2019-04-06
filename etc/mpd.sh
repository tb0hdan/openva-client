#!/bin/bash

MPD=$(which mpd)

case $1 in
    "start")
        action=$1
        ;;
    "stop")
        action=$1
        ;;
    *)
        echo "Unknown action $1"
        exit
        ;;
esac

shift

case $1 in
    "player")
        case ${action} in
            "start")
                ${MPD} etc/mpd_player.conf
                ;;
            "stop")
                kill -9 $(cat /tmp/mpd_player.pid)
                rm -f /tmp/mpd_player.pid
                ;;
        esac
        ;;
    "voice")
        case ${action} in
            "start")
                ${MPD} etc/mpd_voice.conf
                ;;
            "stop")
                kill -9 $(cat /tmp/mpd_voice.pid)
                rm -f /tmp/mpd_voice.pid
                ;;
        esac
        ;;
    *)
        echo "Unknown target $1"
        exit
        ;;
esac
