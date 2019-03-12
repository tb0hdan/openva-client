// includes code from
// https://github.com/mattetti/google-speech/blob/master/cmd/livecaption/main.go
// https://github.com/brentnd/go-snowboy/blob/master/example/listen.go
// https://github.com/pahanini/go-grpc-bidirectional-streaming-example/blob/master/src/server/server.go
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"context"

	"github.com/tb0hdan/openva-server/api"
	"google.golang.org/grpc"

	"github.com/brentnd/go-snowboy"

	"github.com/gordonklaus/portaudio"
	"github.com/shirou/gopsutil/host"
)

const (
	OpenVAServerAddress = "localhost:50001"
	TTSWebServerAddress = "localhost:50005"
)
var (
	TTSWebServerURL = fmt.Sprintf("http://%s/tts/", TTSWebServerAddress)
	TTSCacheDirectory = path.Join("cache", "tts")
	micStopCh = make(chan bool, 1)
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
			fmt.Printf("Result: %+v\n", result)
			for _, msg := range result.Alternatives {
				commands <- msg.Transcript
			}
			breakRequested = true
			break
		}
	}
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
			fmt.Println("handing off the mic")
			return
		default:
		}
		fmt.Print(".")
	}
}

func RunRecognition(micStopCh chan bool, commands chan string, mic *Sound, client api.OpenVAServiceClient) {
	stream, err := client.STT(context.Background())
	if err != nil {
		log.Fatalf("openn stream error %v", err)

	}

	ctx := stream.Context()
	go func() {

		micPoll(stream, micStopCh, mic)

		if err := stream.CloseSend(); err != nil {
			log.Println(err)
		}
	}()

	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		close(micStopCh)
	}()

	streamReader(stream, micStopCh, commands)

}

func sysExit(s string) {
	fmt.Println("received signal", s, "shutting down")
	micStopCh <- true
	os.Exit(0)
}

func HeartBeat(client api.OpenVAServiceClient, player *Player) {
	heartbeatExit := make(chan bool)
	v, _ := host.Info()
	stream, err := client.HeartBeat(context.Background())
	if err != nil {
		fmt.Println("Heartbeat stopped...", err)
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
					SystemUUID: v.HostID,
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
		log.Println(resp)
	}
	<-heartbeatExit
	log.Println("Hearbeat exit")
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

func main() {
	v, _ := host.Info()
	fmt.Println("System UUID: ", v.HostID)
	// Set up a connection to the server.
	conn, err := grpc.Dial(OpenVAServerAddress, grpc.WithInsecure())
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
	signal.Notify(sig, os.Interrupt, os.Kill)
	go func() {
		select {
		case s := <-sig:
			sysExit(s.String())
		}
	}()

	dispatcher := &Dispatcher{
		OpenVAServiceClient: client,
		Player: player,
		Voice: voice,
		Commands: commands,
	}

	// connect to the audio drivers
	portaudio.Initialize()
	defer portaudio.Terminate()

	go dispatcher.Run()
	go HeartBeat(client, player)

	// open the mic
	mic := &Sound{}
	mic.Init()
	defer mic.Close()

	// open the snowboy detector
	d := snowboy.NewDetector("./resources/common.res")
	defer d.Close()


	// set the handlers
	d.HandleFunc(snowboy.NewHotword("./resources/alexa_02092017.umdl", 0.5), func(string) {
		fmt.Println("You said the hotword!")
		player.Pause()
		RunRecognition(micStopCh, commands, mic, client)
	})

	d.HandleSilenceFunc(1*time.Second, func(string) {
		fmt.Println("Silence detected.")
	})

	// display the detector's expected audio format
	sr, nc, bd := d.AudioFormat()
	fmt.Printf("sample rate=%d, num channels=%d, bit depth=%d\n", sr, nc, bd)

	// start detecting using the microphone
	d.ReadAndDetect(mic)
}
