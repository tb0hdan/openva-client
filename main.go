// includes code from
// https://github.com/mattetti/google-speech/blob/master/cmd/livecaption/main.go
// https://github.com/brentnd/go-snowboy/blob/master/example/listen.go
// https://github.com/pahanini/go-grpc-bidirectional-streaming-example/blob/master/src/server/server.go
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/brentnd/go-snowboy"
	"github.com/gordonklaus/portaudio"

	"context"

	"github.com/tb0hdan/openva-server/api"
	"google.golang.org/grpc"

	"github.com/shirou/gopsutil/host"
)

const (
	OpenVAServerAddress = "localhost:50001"
	TTSWebServerAddress = "localhost:50005"
)

// Global vars for versioning
var (
	Build      = "Not available"
	BuildDate  = "Not available"
	GoVersion  = "Not available"
	Version    = "Not available"
	ProjectURL = "Not available"
)

var (
	TTSWebServerURL   = fmt.Sprintf("http://%s/tts/", TTSWebServerAddress)
	TTSCacheDirectory = path.Join("cache", "tts")
	CLI               = flag.Bool("cli", false, "CLI Only mode")
	UUID              string
)

func streamReader(stream api.OpenVAService_STTClient, micCh chan bool, commands chan string) {
	breakRequested := false
	for {
		if breakRequested {
			break
		}
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Cannot stream results: %v", err)
			break
		}

		for _, result := range resp.Results {
			log.Printf("Result: %+v\n", result)
			for _, msg := range result.Alternatives {
				commands <- msg.Transcript
			}
			breakRequested = true
			break
		}
	}
	// Race condition
	micCh <- true
}

func micPoll(stream api.OpenVAService_STTClient, micCh chan bool, mic *Sound) {
	var (
		bufWriter bytes.Buffer
		dummy     []byte
	)

	for {
		bufWriter.Reset()

		mic.Read(dummy)

		binary.Write(&bufWriter, binary.LittleEndian, mic.data)

		err := stream.Send(&api.STTRequest{STTBuffer: bufWriter.Bytes()})
		if err != nil {
			log.Printf("Could not send audio: %v", err)
		}

		select {
		case <-micCh:
			log.Println("handing off the mic")
			return
		default:
		}
		log.Print(".")
	}
}

func RunRecognition(commands chan string, mic *Sound, client api.OpenVAServiceClient) {
	micCh := make(chan bool, 1)

	runDuration := 5 * time.Second
	bgctx := context.Background()
	ctx, _ := context.WithDeadline(bgctx, time.Now().Add(runDuration))
	// max runDuration
	stream, err := client.STT(ctx)
	if err != nil {
		log.Fatalf("openn stream error %v", err)

	}

	ctx = stream.Context()
	go func() {

		micPoll(stream, micCh, mic)

		if err := stream.CloseSend(); err != nil {
			log.Println("Recognition poll ", err)
		}
		log.Println("Recognition poll exit")
	}()

	go func() {
		select {
		case <-ctx.Done():
			break
		case <-micCh:
			break
		}
		if err := ctx.Err(); err != nil {
			log.Println("Recognition ctx ", err)
		}
		micCh <- true
		// Race condition
		// close(micCh)
		log.Println("Recognition ctx exit")
	}()

	streamReader(stream, micCh, commands)
	log.Println("Recognition exit...")
}

func sysExit(s string) {
	log.Println("received signal", s, "shutting down")
	os.Exit(0)
}

func HeartBeat(client api.OpenVAServiceClient, player *Player) {
	log.Println("Heartbeat started...")
	heartbeatExit := make(chan bool)
	v, _ := host.Info()
	stream, err := client.HeartBeat(context.Background())
	if err != nil {
		log.Println("Heartbeat stopped...", err)
		return
	}
	ctx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				log.Println(err)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				heartbeatExit <- true
				break
			case <-heartbeatExit:
				break
			default:
			}

			err = stream.Send(&api.HeartBeatMessage{
				SystemInformation: &api.SystemInformationMessage{
					SystemUUID:       v.HostID,
					UptimeSinceEpoch: time.Now().Unix(),
				},
				PlayerState: &api.PlayerStateMessage{
					NowPlaying: player.NowPlaying,
				},
			})
			time.Sleep(10 * time.Second)
		}

	}()
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			heartbeatExit <- true
			break
		}
		if err != nil {
			heartbeatExit <- true
			log.Println("Cannot stream results: %v", err)
			break
		}
		_ = resp
		//log.Println(resp)
	}
	<-heartbeatExit
	log.Println("Hearbeat exit")
}

func HeartBeatLoop(client api.OpenVAServiceClient, player *Player) {
	i := 1
	for {
		HeartBeat(client, player)
		time.Sleep(time.Duration(i) * 3 * time.Second)
		if i > 40 { // Max - 120s sleep
			i = 1
		}
		i++
	}
}

func TTSWebServer(address string) {
	handler := http.NewServeMux()

	fs := http.FileServer(http.Dir(TTSCacheDirectory))

	handler.Handle("/tts/", http.StripPrefix("/tts/", fs))

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Nothing here")
	})

	srv := http.Server{Addr: address, Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("failed to serve http: %v", err)
		}
	}()
}

func RecognitionMode(player *Player, commands chan string, client api.OpenVAServiceClient) {

	// connect to the audio drivers
	portaudio.Initialize()
	defer portaudio.Terminate()

	// open the mic
	mic := &Sound{}
	//mic.Init()
	mic.InitLastDevice()
	defer mic.Close()

	// open the snowboy detector
	d := snowboy.NewDetector("./resources/common.res")
	defer d.Close()

	// set the handlers
	d.HandleFunc(snowboy.NewHotword("./resources/alexa_02092017.umdl", 0.5), func(string) {
		log.Println("You said the hotword!")
		player.Pause()
		RunRecognition(commands, mic, client)
		player.Resume()
	})

	d.HandleSilenceFunc(1*time.Second, func(string) {
		log.Println("Silence detected.")
	})

	// display the detector's expected audio format
	sr, nc, bd := d.AudioFormat()
	log.Printf("sample rate=%d, num channels=%d, bit depth=%d\n", sr, nc, bd)

	// start detecting using the microphone
	d.ReadAndDetect(mic)
}

func main() {
	flag.Parse()

	v, _ := host.Info()
	UUID = v.HostID

	discoveredOpenVAServer, err := DiscoverOpenVAServers()
	if err != nil || len(discoveredOpenVAServer) == 0 {
		discoveredOpenVAServer = OpenVAServerAddress
	}

	// Set up a connection to the server.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	kp := keepalive.ClientParameters{
		Time:    20 * time.Second,
		Timeout: 25 * time.Second,
	}
	bc := grpc.BackoffConfig{
		MaxDelay: 120 * time.Second,
	}
	conn, err := grpc.DialContext(
		ctx, discoveredOpenVAServer,
		grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithKeepaliveParams(kp), grpc.WithBackoffConfig(bc),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// Player MPD
	player := &Player{}
	player.Connect("localhost:6600")
	go player.NowPlayingUpdater()
	defer player.Close()
	//

	// Voice MPD
	voice := &Player{}
	voice.Connect("localhost:6601")
	defer voice.Close()
	//

	// Start TTS Web server
	go TTSWebServer(TTSWebServerAddress)

	client := api.NewOpenVAServiceClient(conn)

	// Clean shutdown (theoretically)
	sig := make(chan os.Signal, 1)
	commands := make(chan string)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-sig
		sysExit(s.String())

	}()

	dispatcher := &Dispatcher{
		OpenVAServiceClient: client,
		Player:              player,
		Voice:               voice,
		Commands:            commands,
		HostInfo:            v,
	}

	go dispatcher.Run()
	go HeartBeatLoop(client, player)

	if !*CLI {
		RecognitionMode(player, commands, client)
	} else {
		RunCLI(commands)
	}
}
