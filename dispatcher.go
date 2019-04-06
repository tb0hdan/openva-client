package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/shirou/gopsutil/host"

	"github.com/tb0hdan/openva-server/api"
)

// Reusable utilities
func volumeToMPDVolume(volume int) (int, error) {
	if volume < 0 || volume > 10 {
		return 0, errors.New("0 < volume < 10")
	}
	return -100 + volume*20, nil
}

func ParseVolume(cmd string, player *Player) { // nolint gocyclo
	volume := 0
	split := strings.Split(cmd, " ")
	if len(split) != 2 {
		return
	}
	switch strings.ToLower(split[1]) {
	case "one":
		volume = 1
	case "two":
		volume = 2
	case "three":
		volume = 3
	case "four":
		volume = 4
	case "five":
		volume = 5
	case "six":
		volume = 6
	case "seven":
		volume = 7
	case "eight":
		volume = 8
	case "nine":
		volume = 9
	case "ten":
		volume = 10
	default:
		return
	}
	mpdVolume, err := volumeToMPDVolume(volume)
	if err != nil {
		log.Println("error setting volume")
	}
	player.SetVolume(mpdVolume)
}

type Dispatcher struct {
	OpenVAServiceClient api.OpenVAServiceClient
	Player              *Player
	Voice               *Player
	Commands            <-chan string
	HostInfo            *host.InfoStat
	ClientConfig        *api.ClientConfigMessage
}

func (d *Dispatcher) SayFile(text string) string {
	var err error

	cachedFile := path.Join(TTSCacheDirectory, strings.ToLower(strings.Replace(text, " ", "_", -1)+".mp3"))
	_, err = os.Open(cachedFile)
	if os.IsExist(err) {
		return cachedFile
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := d.OpenVAServiceClient.TTSStringToMP3(ctx, &api.TTSRequest{Text: text})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		log.Println("could not create tempfile", err)
	}
	ioutil.WriteFile(f.Name(), r.MP3Response, 0644)
	os.Rename(f.Name(), cachedFile)

	return cachedFile

}

func (d *Dispatcher) Say(text string) {

	d.Player.SetVolume(3)
	defer d.Player.SetVolume(10)
	cachedFile := d.SayFile(text)
	base := path.Base(cachedFile)

	url := TTSWebServerURL + base
	d.Voice.PlayURL(url)
}

func (d *Dispatcher) HandleServerSideCommand(cmd string) {
	reply, err := d.OpenVAServiceClient.HandleServerSideCommand(context.Background(), &api.TTSRequest{Text: cmd})
	if err != nil {
		d.Say(d.ClientConfig.Locale.CouldNotUnderstandMessage)
		return
	}
	if reply.IsError && len(reply.TextResponse) > 0 {
		d.Say(reply.TextResponse)
		return
	}

	if len(reply.TextResponse) > 0 {
		d.Say(reply.TextResponse)
	}

	if len(reply.Items) > 0 {
		urls := make([]string, 0)
		for _, item := range reply.Items {
			urls = append(urls, item.URL)
		}
		d.Player.ShuffleURLList(urls)
	}
}

func (d *Dispatcher) Run() { // nolint gocyclo
	var err error

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	d.ClientConfig, err = d.OpenVAServiceClient.ClientConfig(ctx, &api.ClientMessage{
		SystemUUID: d.HostInfo.HostID,
	})
	if err != nil {
		log.Fatalf("Could not get configuration: %+v", err)
	}

	for cmd := range d.Commands {
		cmd = strings.TrimSpace(cmd)
		first := strings.ToLower(strings.Split(cmd, " ")[0])
		switch first {
		// Basic controls
		case d.ClientConfig.Locale.VolumeMessage:
			ParseVolume(cmd, d.Player)
		case d.ClientConfig.Locale.PauseMessage:
			d.Player.Pause()
		case d.ClientConfig.Locale.ResumeMessage:
			d.Player.Resume()
		case d.ClientConfig.Locale.StopMessage:
			d.Player.Stop()
		case d.ClientConfig.Locale.NextMessage:
			d.Player.Next()
		case d.ClientConfig.Locale.PreviousMessage:
			d.Player.Previous()
		case d.ClientConfig.Locale.RebootMessage:
			d.Say("Rebooting")
			sysExit("USER_EXIT_REQ")
		// Timers and Alarms
		case "set":
			timerLength := strings.TrimPrefix(cmd, first)
			d.Say(fmt.Sprintf("%s timer starting now", timerLength))
		case "wake":
			alarmWhen := strings.TrimPrefix(cmd, "wake me up at")
			d.Say(fmt.Sprintf("Alarm set for %s tomorrow", alarmWhen))
		case "cancel":
			if strings.Contains(cmd, "timer") {
				d.Say("Canceling all your timers")
			}
			if strings.Contains(cmd, "alarm") {
				d.Say("Canceling all your alarms")
			}
		// Processed by server
		default:
			go d.HandleServerSideCommand(cmd)
		}
	}
}
