package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tb0hdan/gompd-transition/v2/mpd"
)

const (
	Playing = "play"
	Paused  = "pause"
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
	_ = p.Conn.Add(url)
}

func (p *Player) Play(pos int) {
	_ = p.Conn.Play(pos)
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
	for _, playlistURL := range urlList {
		p.Add(playlistURL)
	}
	p.Play(0)
}

func (p *Player) ShuffleURLList(urlList []string) {
	p.Clear()
	for _, playlistURL := range urlList {
		p.Add(playlistURL)
	}
	p.Shuffle()
	p.Play(0)
}

func (p *Player) Pause() {
	state, err := p.State()
	if err != nil {
		log.Fatalln(err)
	}
	if state == Playing {
		p.Paused = true
		_ = p.Conn.Pause(p.Paused)
	}
}

func (p *Player) Resume() {
	state, err := p.State()
	if err != nil {
		log.Fatalln(err)
	}
	if state != Playing {
		p.Paused = false
		_ = p.Conn.Pause(p.Paused)
	}
}

func (p *Player) SetVolume(volume int) {
	_ = p.Conn.SetVolume(volume)
}

func (p *Player) GetVolume() (volume int) {
	status, err := p.Conn.Status()
	if err != nil {
		log.Fatal(err)
	}
	// Some systems do not allow MPD to get/set volume
	volumeString := status["volume"]
	if len(volumeString) == 0 {
		return 0
	}
	vol, err := strconv.Atoi(status["volume"])
	if err != nil {
		log.Fatal(err)
	}
	return vol
}

func (p *Player) Clear() {
	_ = p.Conn.Clear()
}

func (p *Player) Next() {
	_ = p.Conn.Next()
}

func (p *Player) Previous() {
	_ = p.Conn.Previous()
}

func (p *Player) Stop() {
	_ = p.Conn.Stop()
}

func (p *Player) Shuffle() {
	_ = p.Conn.Shuffle(0, -1)
}

func (p *Player) State() (state string, err error) {
	status, err := p.Conn.Status()
	return status["state"], err

}

func (p *Player) NowPlayingUpdater() { // nolint gocyclo
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
		if status["state"] == Playing {
			if song["Artist"] != "" {
				line1 = fmt.Sprintf("%s - %s", song["Artist"], song["Title"])
			} else {
				line1 = song["Title"]
			}
			// URL
			if len(line1) == 0 {
				artist, _, track := URLToTrack(song["file"])
				if len(artist) > 0 && len(track) > 0 {
					line1 = fmt.Sprintf("%s - %s", artist, track)
				}
			}
		} else if status["state"] == Paused {
			line = ""
			line1 = ""
			p.NowPlaying = ""
		}
		if line != line1 {
			line = line1
			p.NowPlaying = line
		}
		p.Volume = p.GetVolume()
		time.Sleep(time.Second)
	}
}

func Normalize(entity string, regexes [][]string) (result string) {
	result = entity
	for _, regularExpression := range regexes {
		re := regexp.MustCompile(regularExpression[0])
		result = re.ReplaceAllString(result, regularExpression[1])
	}
	result = strings.TrimSpace(result)
	return
}

func NormalizeTrack(track string) string {
	regexes := [][]string{
		{`^[0-9]+\.\s+`, ``}, {`^[0-9]+\s`, ``}, {`^[0-9]+\-[0-9]+\s`, ``},
		{`\.mp3$`, ``}, {`\.wmv$`, ``},
		{`\(Official\sMusic\sVideo\)$`, ``}, {`\(No\sLyrics\)`, ``}, {`\s?\(?Official\sVideo\)?$`, ``},
		{`\s?\[?Official\sMusic\sVideo\]?$`, ``},
		{`\s?\[?Lyrics\]?$`, ``},
		{`\s?lyrics$`, ``},
		{`_`, " "}, {`^-\s`, ``},
		{`&amp;`, "&"},
		{`&#39;`, "'"},
		{`on_t`, `on't'`}, // D?on't
		{`_$`, ``},        // HOW WOULD YOU FEEL_ - David Morales
		{`\(?zaycev.net\)?`, ``},
		{`\s?\[?HQ\]?$`, ``},
		{`\s?\(?HD\)?$`, ``},
		{`\s+\[?Ultra\sMusic\]?$`, ``},
		{`(FULL)?\s?HD$`, ``},
		{`\s?\(?Official\sVideo\s?HD\)?$`, ``},
		{`\s?\[?OUT\sNOW\]?$`, ``},
		{`\s\(?Lyric\sVideo\)?$`, ``},
		{`\s?Promo\sVideo$`, ``},
	}
	return Normalize(track, regexes)
}

func NormalizeArtist(artist string) string {
	regexes := [][]string{
		{`\[mp3ex\.net\]`, ""},
		{`\(Official\sVideo\)`, ""},
		{`_`, "/"},
		{`&amp;`, "&"},
		{`&#39;`, "'"},
	}
	return Normalize(artist, regexes)
}

func URLToTrack(urlValue string) (artist, album, track string) {
	parsedURL, err := url.Parse(urlValue)
	if err != nil {
		log.Printf("Could not parse url: %s - %+v", urlValue, err)
		return
	}
	// only valid for our library
	if !strings.HasPrefix(parsedURL.Path, "/music/") {
		return
	}
	processablePath := strings.TrimPrefix(parsedURL.Path, "/music/")

	splitPath := strings.Split(processablePath, "/")
	// Artist / Album / Track
	if len(splitPath) == 3 {
		artist = NormalizeArtist(strings.TrimSpace(splitPath[0]))
		album = splitPath[1]
		track = NormalizeTrack(strings.TrimSpace(splitPath[2]))
		// special case, "/music/xxx" directory with just tracks in it
	} else if len(splitPath) == 2 {
		splitTrack := strings.Split(splitPath[1], " - ")
		if len(splitTrack) == 2 {
			artist = NormalizeArtist(strings.TrimSpace(splitTrack[0]))
			track = NormalizeTrack(strings.TrimSpace(splitTrack[1]))
		} else {
			splitTrack = strings.Split(splitPath[1], " ━ ")
			if len(splitTrack) == 2 {
				artist = NormalizeArtist(strings.TrimSpace(splitTrack[0]))
				track = NormalizeTrack(strings.TrimSpace(splitTrack[1]))
			}
		}

	}

	return artist, album, track
}
