// slightly modified version of https://github.com/Kitt-AI/snowboy/blob/master/examples/Go/listen/main.go
package main

import (
	"bytes"
	"encoding/binary"
	"log"

	"github.com/gordonklaus/portaudio"
)

// Sound represents a sound stream implementing the io.Reader interface
// that provides the microphone data.
type Sound struct {
	stream *portaudio.Stream
	data   []int16
}

const (
	InputChannels  = 1
	OutputChannels = 0
	SampleRate     = 16000
	BufferLength   = 1000
)

// Init initializes the Sound's PortAudio stream.
func (s *Sound) Init() {

	s.data = make([]int16, BufferLength)

	// initialize the audio recording interface
	err := portaudio.Initialize()
	if err != nil {
		log.Printf("error initializing audio interface: %s", err)
		return
	}

	// open the sound input stream for the microphone
	stream, err := portaudio.OpenDefaultStream(InputChannels, OutputChannels, float64(SampleRate), len(s.data), s.data)
	if err != nil {
		log.Printf("error open default audio stream: %s", err)
		return
	}

	err = stream.Start()
	if err != nil {
		log.Printf("error on stream start: %s", err)
		return
	}

	s.stream = stream
}

func (s *Sound) InitLastDevice() {

	var (
		lastDevice *portaudio.DeviceInfo
		emptyDev   *portaudio.DeviceInfo
	)

	s.data = make([]int16, BufferLength)

	// initialize the audio recording interface
	err := portaudio.Initialize()
	if err != nil {
		log.Printf("error initializing audio interface: %s", err)
		return
	}

	devices, err := portaudio.Devices()
	if err != nil {
		log.Printf("Get devices failed: %v", err)
		return
	}

	for _, device := range devices {
		// skip output-only devices
		if device.MaxInputChannels == 0 {
			continue
		}
		log.Println(device)
		lastDevice = device
	}

	log.Printf("Using `%s` as input...", lastDevice.Name)

	streamParams := portaudio.HighLatencyParameters(lastDevice, emptyDev)
	streamParams.Input.Channels = InputChannels
	streamParams.Output.Channels = OutputChannels
	streamParams.SampleRate = float64(SampleRate)
	streamParams.FramesPerBuffer = len(s.data)
	stream, err := portaudio.OpenStream(streamParams, s.data)
	if err != nil {
		log.Printf("error open custom audio stream: %s", err)
		return
	}

	err = stream.Start()
	if err != nil {
		log.Printf("error on stream start: %s", err)
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
