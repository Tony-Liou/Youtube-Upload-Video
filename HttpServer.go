package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"myUpload"
)

type frame struct {
	Streaming bool   `json:"streaming"` // isStreaming
	Status    int    `json:"status"`
	StreamID  string `json:"streamId"`
}

type indexHandler struct {
	frame frame
}

func (ih indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		for k, v := range r.URL.Query() {
			fmt.Printf("%s: %s\n", k, v)
		}
	case http.MethodPost:
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%s\n", reqBody)

		if err = json.Unmarshal(reqBody, &ih.frame); err == nil {
			fmt.Print(ih.frame)
		}

		if ih.frame.Streaming {
			doSomething(ih.frame.StreamID)
		}

		w.Write([]byte("Recieved a POST request\n"))
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// Dump RTMP streaming from 17 live
func execStreamlink(StreamID string) {
	curTime := time.Now().Format("_2006-01-02_15-04-05")

	//fmt.Println(curTime)
	app := "streamlink"

	option := "-o"
	filename := StreamID + curTime + ".flv"
	url := "17.live/live/" + StreamID
	quality := "best"

	cmd := exec.Command(app, option, filename, url, quality)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}
	fmt.Println("Result: " + out.String())
}

func doSomething(streamID string) {
	execStreamlink(streamID)
	myUpload.UploadVideo()
}

func main() {
	port := ":8080"
	var myHandler indexHandler
	srv := &http.Server{
		Addr:    port,
		Handler: myHandler,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We recieved an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	fmt.Println("Starting server... Port is ", port)
	err := srv.ListenAndServe()
	if err != nil {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
