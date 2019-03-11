package main

import (
	"github.com/tb0hdan/gompd/mpd"
	log "github.com/sirupsen/logrus"
)

type Player struct {
	Conn   *mpd.Client
	Volume int
	Paused bool
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

func (p *Player) Stop() {
	p.Conn.Stop()
}

func (p *Player) Shuffle() {
	p.Conn.Shuffle(0, -1)
}
