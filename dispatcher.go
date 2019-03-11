package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/tb0hdan/openva-server/api"
	"io/ioutil"
	"log"
	mp3player "openva-client/player"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

func SayFile(text string, client api.OpenVAServiceClient) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := client.TTSStringToMP3(ctx, &api.TTSRequest{Text: text})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	f, err := ioutil.TempFile("", "")

	fileName := f.Name()
	ioutil.WriteFile(fileName, r.MP3Response, 0644)
	return fileName

}
func Say(text string, client api.OpenVAServiceClient) {
	var err error
	cacheDir := path.Join("cache", "tts")
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		fmt.Println("Cache dir exists")
	}

	cachedFile := path.Join(cacheDir, strings.ToLower(strings.Replace(text, " ", "_", -1)+".mp3"))
	_, err = os.Open(cachedFile)
	if os.IsNotExist(err) {
		fname := SayFile(text, client)
		os.Rename(fname, cachedFile)
	}

	mp3player.Play(cachedFile)
}

func ShuffleLibraryWithCriteria(client api.OpenVAServiceClient, player *Player, criteria string) {
	items, err := client.Library(context.Background(), &api.LibraryFilterRequest{Criteria: criteria})
	if err != nil {
		log.Fatal(err)
	}
	urls := make([]string, 0)
	for _, item := range items.Items {
		urls = append(urls, item.URL)
	}
	player.ShuffleURLList(urls)
}

func ShuffleLibrary(client api.OpenVAServiceClient, player *Player) {
	ShuffleLibraryWithCriteria(client, player, "")
}

func PlayParser(cmd string, client api.OpenVAServiceClient, player *Player) {
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
	ShuffleLibraryWithCriteria(client, player, what)
}

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

func HandleServerSideCommand(cmd string, client api.OpenVAServiceClient) {
	reply, err := client.HandlerServerSideCommand(context.Background(), &api.TTSRequest{Text: cmd})
	if err != nil {
		Say("I could not understand you", client)
		return
	}
	f, err := ioutil.TempFile("", "")
	defer os.Remove(f.Name())
	ioutil.WriteFile(f.Name(), reply.MP3Response, 0644)
	mp3player.Play(f.Name())

}

func commandDispatcher(commands <-chan string, client api.OpenVAServiceClient, player *Player) {
	for cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		fmt.Println(cmd)
		first := strings.ToLower(strings.Split(cmd, " ")[0])
		switch first {
		case "play":
			go PlayParser(cmd, client, player)
		// Basic controls
		case "volume":
			ParseVolume(cmd, player)
		case "pause":
			player.Pause()
		case "resume":
			player.Resume()
		case "stop":
			player.Stop()
		case "next":
			player.Next()
		// Shuffle whole library
		case "shuffle":
			Say("Shuffling your library", client)
			go ShuffleLibrary(client, player)
		case "reboot":
			Say("Rebooting", client)
			sysExit("USER_EXIT_REQ")
		default:
			go HandleServerSideCommand(cmd, client)
		}
	}
}
