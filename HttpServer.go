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
	"strconv"
	"time"

	"github.com/Tony-Liou/Youtube-Upload-Video/myUpload"
)

var isDownloading bool // Streamlink is dumping the stream

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
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GET"))
	case http.MethodPost:
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		fmt.Printf("Request body=%s\n", reqBody)

		if err = json.Unmarshal(reqBody, &ih.frame); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Println(err)
			return
		}
		fmt.Println(ih.frame)

		if ih.frame.IsStreaming && !isDownloading {
			go processStreaming(ih.frame.StreamID)
		}

		msg := fmt.Sprintf("%v is streaming: %v\n", ih.frame.StreamID, ih.frame.IsStreaming)
		w.Write([]byte(msg))
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// Dump RTMP streaming from 17 live, and return a current time string and a filename (with path)
func execStreamlink(StreamID string) (string, string) {
	t := time.Now()
	curTime := t.Format("2006/01/02_15:04:05_Mon")
	curTimeStamp := strconv.FormatInt(t.Unix(), 10)

	app := "streamlink"

	option := "-o"
	filename := StreamID + "_" + curTimeStamp + ".flv"
	url := "17.live/live/" + StreamID
	quality := "best"

	cmd := exec.Command(app, option, filename, url, quality)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	log.Println("Streamlink starting...")
	err := cmd.Run()
	if err != nil {
		log.Println(fmt.Sprint(err) + ": " + stderr.String())
	}
	log.Println("Result: ", out.String())

	return curTime, filename
}

// Execute shell to remove the video file
func removeFile(path string) {
	err := exec.Command("rm", path).Run()
	if err != nil {
		log.Println(err)
	}
}

// Executing streamlink to dump the live streaming.
// After the live streaming ending, upload the video to Youtube.
func processStreaming(streamID string) {
	log.Println("Processing streaming...")

	isDownloading = true
	time, uri := execStreamlink(streamID)
	isDownloading = false

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

	sendVideoInfo(videoID)
}

func sendVideoInfo(videoID string) {
	url := `address` // TODO change this
	url += `?videoID=` + videoID

	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Status code: %v, status: %v\n", resp.StatusCode, resp.Status)
	}
}

func main() {
	f, err := os.OpenFile("myServer.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

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

	log.Println("Starting server... Port is ", port)
	err = srv.ListenAndServe()
	if err != nil {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
