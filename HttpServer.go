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

	"github.com/Tony-Liou/Youtube-Upload-Video/myUpload"
)

type frame struct {
	IsStreaming bool   `json:"isStreaming"`
	StreamID    string `json:"streamId"`
	VideoStatus string `json:"videoStatus"` // public, unlisted, private
}

type indexHandler struct {
	frame frame
}

func (ih indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// for k, v := range r.URL.Query() {
		// 	fmt.Printf("%s: %s\n", k, v)
		// }
		w.Write([]byte("Get\n"))
	case http.MethodPost:
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
		}

		fmt.Printf("Request body=%s\n", reqBody)

		if err = json.Unmarshal(reqBody, &ih.frame); err == nil {
			log.Println(ih.frame)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Need json data\n" + err.Error()))
		}

		if ih.frame.IsStreaming {
			go processStreaming(ih.frame.StreamID)
		}

		msg := fmt.Sprintf("isStreaming: %v\n", ih.frame.IsStreaming)
		w.Write([]byte(msg))
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// Dump RTMP streaming from 17 live, and return a current time string and a filename (with path)
func execStreamlink(StreamID string) (string, string) {
	curTime := time.Now().Format("_2006-01-02_15-04-05_Mon")

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
		log.Println(fmt.Sprint(err) + ": " + stderr.String())
		//return curTime, ""
	}
	fmt.Println("Result: " + out.String())

	return curTime, filename
}

// Execute shell to remove the video file
func removeFile(path string) {
	cmd := exec.Command("rm", path)

	err := cmd.Run()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("File deleted. ", path)
	}
}

// Executing streamlink to dump the live streaming.
// After the live streaming ending, upload the video to Youtube.
func processStreaming(streamID string) {
	log.Println("Processing streaming...")

	time, uri := execStreamlink(streamID)

	setting := &myUpload.VideoSetting{
		Filename:    uri,
		Title:       time,
		Description: "https://17.live/live/" + streamID,
		Category:    "22",
		Keywords:    "17Live," + streamID,
		Privacy:     "unlisted",
	}
	videoID := myUpload.UploadVideo(setting)
	if videoID == "" {
		log.Println("Upload video failed.")
		return
	}
	log.Println("Video uploaded. ID: ", videoID)

	removeFile(uri)
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
