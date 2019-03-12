package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/tb0hdan/openva-server/api"
)

// Reusable utilities
func volumeToMPDVolume(volume int) (int, error) {
	if volume < 0 || volume > 10 {
		return 0, errors.New("0 < volume < 10")
	}
	return -100 + volume*20, nil
}

func ParseVolume(cmd string, player *Player) {
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

//

type Dispatcher struct {
	OpenVAServiceClient api.OpenVAServiceClient
	Player              *Player
	Voice               *Player
	Commands            <-chan string
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

	ioutil.WriteFile(f.Name(), r.MP3Response, 0644)
	os.Rename(f.Name(), cachedFile)

	return cachedFile

}

func (d *Dispatcher) Say(text string) {
	cachedFile := d.SayFile(text)
	base := path.Base(cachedFile)
	url := TTSWebServerURL + base
	d.Voice.PlayURL(url)
}

func (d *Dispatcher) ShuffleLibraryWithCriteria(criteria string) {
	items, err := d.OpenVAServiceClient.Library(context.Background(), &api.LibraryFilterRequest{Criteria: criteria})
	if err != nil {
		log.Fatal(err)
	}
	urls := make([]string, 0)
	for _, item := range items.Items {
		urls = append(urls, item.URL)
	}
	d.Player.ShuffleURLList(urls)
}

func (d *Dispatcher) ShuffleLibrary() {
	d.ShuffleLibraryWithCriteria("")
}

func (d *Dispatcher) PlayParser(cmd string) {
	var what string
	re := regexp.MustCompile(`^play (.*) from my library`)
	submatch := re.FindStringSubmatch(strings.ToLower(cmd))
	if re.MatchString(cmd) && len(submatch) > 1 {
		what = strings.TrimSpace(submatch[1])
	}
	if len(what) == 0 {
		// FIXME: Extend here (or at least say something)
		return
	}
	d.ShuffleLibraryWithCriteria(what)
}

func (d *Dispatcher) HandleServerSideCommand(cmd string) {
	reply, err := d.OpenVAServiceClient.HandlerServerSideCommandText(context.Background(), &api.TTSRequest{Text: cmd})
	if err != nil {
		d.Say("I could not understand you")
		return
	}
	d.Say(reply.Text)
}

func (d *Dispatcher) Run() {
	for cmd := range d.Commands {
		cmd = strings.TrimSpace(cmd)
		fmt.Println(cmd)
		first := strings.ToLower(strings.Split(cmd, " ")[0])
		switch first {
		case "play":
			go d.PlayParser(cmd)
		// Basic controls
		case "volume":
			ParseVolume(cmd, d.Player)
		case "pause":
			d.Player.Pause()
		case "resume":
			d.Player.Resume()
		case "stop":
			d.Player.Stop()
		case "next":
			d.Player.Next()
		// Shuffle whole library
		case "shuffle":
			d.Say("Shuffling your library")
			go d.ShuffleLibrary()
		case "reboot":
			d.Say("Rebooting")
			sysExit("USER_EXIT_REQ")
		default:
			go d.HandleServerSideCommand(cmd)
		}
	}
}
