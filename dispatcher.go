package main

import (
	"context"
	"fmt"
	"github.com/tb0hdan/openva-server/api"
	"io/ioutil"
	"log"
	mp3player "openva-client/player"
	"os"
	"path"
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

func ShuffleLibrary(client api.OpenVAServiceClient, player *Player) {
	items, err := client.Library(context.Background(), &api.LibraryFilterRequest{})
	if err != nil {
		log.Fatal(err)
	}
	urls := make([]string, 0)
	for _, item := range items.Items {
		urls = append(urls, item.URL)
	}
	player.ShuffleURLList(urls)

	}

func commandDispatcher(commands <-chan string, client api.OpenVAServiceClient, player *Player) {
	for cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		fmt.Println(cmd)
		first := strings.ToLower(strings.Split(cmd, " ")[0])
		switch first {
		case "pause":
			player.Pause()
		case "resume":
			player.Resume()
		case "shuffle":
			go ShuffleLibrary(client, player)
		case "reboot":
			sysExit("USER_EXIT_REQ")
		default:
			Say(cmd, client)
			fmt.Println("Unknown command")
		}
	}
}
