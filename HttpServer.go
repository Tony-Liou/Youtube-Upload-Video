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

type messageObjects struct {
	MsgType string `json:"type"`
	Text    string `json:"text"` // max 5000
}

type lineBody struct {
	To                   string           `json:"to"`
	Messages             []messageObjects `json:"messages"` // max 5
	NotificationDisabled bool             `json:"notificationDisabled,omitempty"`
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
			fmt.Println(ih.frame)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Need json data\n" + err.Error()))
		}

		if ih.frame.IsStreaming {
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
	cmd := exec.Command("rm", path)

	err := cmd.Run()
	if err != nil {
		log.Println(err)
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

	lineMsg := fmt.Sprintf("StreamID: %s VoD is uploaded. Video ID: %s\n", streamID, videoID)
	pushLineMsg(lineMsg)
}

// Send Line message to the group chat
func pushLineMsg(msg string) {
	token := `Enter your channel access token`
	url := "https://api.line.me/v2/bot/message/push"

	msgObj := messageObjects{
		MsgType: "text",
		Text:    msg,
	}

	obj := lineBody{
		To: "Ccfc3c76624b68ec16994ed9e9da00d93",
		Messages: []messageObjects{
			msgObj,
		},
	}

	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		log.Printf("JSON marshal failed: %v, Line push message failed", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}

	resp, err := client.Do(req)
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
