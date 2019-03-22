// slightly modified version of https://github.com/Kitt-AI/snowboy/blob/master/examples/Go/listen/main.go
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/gordonklaus/portaudio"
)

// Sound represents a sound stream implementing the io.Reader interface
// that provides the microphone data.
type Sound struct {
	stream *portaudio.Stream
	data   []int16
}

// Init initializes the Sound's PortAudio stream.
func (s *Sound) Init() {
	inputChannels := 1
	outputChannels := 0
	sampleRate := 16000
	s.data = make([]int16, 1024)

	// initialize the audio recording interface
	err := portaudio.Initialize()
	if err != nil {
		fmt.Errorf("error initializing audio interface: %s", err)
		return
	}

	// open the sound input stream for the microphone
	stream, err := portaudio.OpenDefaultStream(inputChannels, outputChannels, float64(sampleRate), len(s.data), s.data)
	if err != nil {
		fmt.Errorf("error open default audio stream: %s", err)
		return
	}

	err = stream.Start()
	if err != nil {
		fmt.Errorf("error on stream start: %s", err)
		return
	}

	s.stream = stream
}

// Close closes down the Sound's PortAudio connection.
func (s *Sound) Close() {
	s.stream.Close()
	portaudio.Terminate()
}

// Read is the Sound's implementation of the io.Reader interface.
func (s *Sound) Read(p []byte) (int, error) {
	s.stream.Read()

	buf := &bytes.Buffer{}
	for _, v := range s.data {
		binary.Write(buf, binary.LittleEndian, v)
	}

	copy(p, buf.Bytes())
	return len(p), nil
}
