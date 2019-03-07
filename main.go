// includes code from
// https://github.com/mattetti/google-speech/blob/master/cmd/livecaption/main.go
// https://github.com/brentnd/go-snowboy/blob/master/example/listen.go
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/tb0hdan/openva-server/api"
	"google.golang.org/grpc"

	"github.com/brentnd/go-snowboy"

	"github.com/gordonklaus/portaudio"
)

const address = "localhost:50001"

var micStopCh = make(chan bool, 1)

func getStream() (stream api.OpenVAService_STTClient){

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	runDuration := 70 * time.Second
	bgctx := context.Background()
	ctx, _ := context.WithDeadline(bgctx, time.Now().Add(runDuration))

	client := api.NewOpenVAServiceClient(conn)
	stream, err = client.STT(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return
}

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

func sysExit(s string) {
	fmt.Println("received signal", s, "shutting down")
	micStopCh <- true
	os.Exit(0)
}

func main() {
	fmt.Println("System UUID: ", GetSysUUID())
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := api.NewOpenVAServiceClient(conn)

	// Clean shutdown
	sig := make(chan os.Signal, 1)
	commands := make(chan string)
	signal.Notify(sig, os.Interrupt, os.Kill)
	go func() {
		select {
		case s := <-sig:
			sysExit(s.String())
		}
	}()

	// connect to the audio drivers
	portaudio.Initialize()
	defer portaudio.Terminate()

	go commandDispatcher(commands, client)

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
		stream := getStream()
		go micPoll(stream, micStopCh, mic)
		streamReader(stream, micStopCh, commands)
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
