package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tb0hdan/gompd-transition/v2/mpd"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Player struct {
	Conn       *mpd.Client
	Volume     int
	Paused     bool
	NowPlaying string
}

func (p *Player) Connect(address string) {
	var err error

	if address == "" {
		address = "localhost:6600"
	}
	// FIXME: Add retry
	p.Conn, err = mpd.Dial("tcp", address)
	if err != nil {
		log.Fatalln(err)
	}
}

func (p *Player) Close() {
	err := p.Conn.Close()
	if err != nil {
		log.Printf("Close errored: %+v", err)
	}
}

func (p *Player) Add(url string) {
	p.Conn.Add(url)
}

func (p *Player) Play(pos int) {
	p.Conn.Play(pos)
}

func (p *Player) PlayURL(url string) {
	// FIXME: Add uses Sprintf and doesn't work with quoted URLs!
	// FIXME: Check errors!
	p.Clear()
	p.Add(url)
	p.Play(0)
}

func (p *Player) PlayURLList(urlList []string) {
	p.Clear()
	for _, url := range urlList {
		p.Add(url)
	}
	p.Play(0)
}

func (p *Player) ShuffleURLList(urlList []string) {
	p.Clear()
	for _, url := range urlList {
		p.Add(url)
	}
	p.Shuffle()
	p.Play(0)
}

func (p *Player) Pause() {
	if p.Paused {
		p.Paused = false
	} else {
		p.Paused = true
	}

	p.Conn.Pause(p.Paused)
}

func (p *Player) Resume() {
	p.Paused = false
	p.Conn.Pause(p.Paused)
}

func (p *Player) SetVolume(volume int) {
	p.Volume = volume
	p.Conn.SetVolume(volume)
}

func (p *Player) GetVolume() (volume int) {
	return p.Volume
}

func (p *Player) Clear() {
	p.Conn.Clear()
}

func (p *Player) Next() {
	p.Conn.Next()
}

func (p *Player) Previous() {
	p.Conn.Previous()
}

func (p *Player) Stop() {
	p.Conn.Stop()
}

func (p *Player) Shuffle() {
	p.Conn.Shuffle(0, -1)
}

func (p *Player) NowPlayingUpdater() {
	line := ""
	line1 := ""
	for {
		status, err := p.Conn.Status()
		if err != nil {
			log.Fatalln(err)
		}
		song, err := p.Conn.CurrentSong()
		if err != nil {
			log.Fatalln(err)
		}
		if status["state"] == "play" {
			if song["Artist"] != "" {
				line1 = fmt.Sprintf("%s - %s", song["Artist"], song["Title"])
			} else {
				line1 = song["Title"]
			}
			// URL
			if len(line1) == 0 {
				artist, _, track := URLToTrack(song["file"])
				if len(artist) >0 && len(track) > 0 {
					line1 = fmt.Sprintf("%s - %s", artist, track)
				}
			}
		}
		if line != line1 {
			line = line1
			p.NowPlaying = line
			fmt.Println(line)
		}
		time.Sleep(1e9)
	}
}

func NormalizeTrack(track string) (result string) {
	result = track
	regexes := []string{`^[0-9]+\s`, `^[0-9]+\-[0-9]+\s`, `\.mp3$`, `\(Official\sMusic\sVideo\)$`}
	for _, reg := range regexes {
		re := regexp.MustCompile(reg)
		result = re.ReplaceAllString(result, "")
	}
	return
}

func URLToTrack(urlValue string) (artist, album, track string) {
	url, err := url.Parse(urlValue)
	if err != nil {
		log.Printf("Could not parse url: %s - %+v", urlValue, err)
		return
	}
	// only valid for our library
	if !strings.HasPrefix(url.Path, "/music/") {
		return
	}
	processablePath := strings.TrimPrefix(url.Path, "/music/")

	splitPath := strings.Split(processablePath, "/")
	// Artist / Album / Track
	if len(splitPath) != 3 {
		return
	}

	artist = splitPath[0]
	album = splitPath[1]
	track = NormalizeTrack(splitPath[2])
	return
}
