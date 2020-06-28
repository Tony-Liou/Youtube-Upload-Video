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
	"strings"
	"time"

	"github.com/Tony-Liou/Youtube-Upload-Video/myUpload"
)

var isDownloading bool       // Streamlink is dumping the stream
var isOnline map[string]bool // key is stream ID and value is online or not

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

		if ih.frame.StreamID != "" {
			isOnline[ih.frame.StreamID] = ih.frame.IsStreaming
			if isOnline[ih.frame.StreamID] && !isDownloading {
				go processStreaming(ih.frame.StreamID)
			}
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
func removeVideoFile(path string) {
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
	recordTime, uri := execStreamlink(streamID)
	isDownloading = false

	// Check the streamer is really offline or just a temporary hang
	time.AfterFunc(time.Minute, func() {
		if !isOnline[streamID] {
			return
		}

		if isActive(streamID) {
			go processStreaming(streamID)
		} else {
			isOnline[streamID] = false
		}
	})

	setting := &myUpload.VideoSetting{
		Filename:    uri,
		Title:       recordTime,
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

	removeVideoFile(uri)

	sendVideoInfo(videoID)
}

func sendVideoInfo(videoID string) {
	url := `{youraddress}` // TODO change this
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

// Check the streamer is online or not
func isActive(streamID string) bool {
	url := `https://api-dsa.17app.co/api/v1/lives/{sID}/viewers/alive`

	url = strings.Replace(url, "{sID}", streamID, 1)

	tmpMap := make(map[string]string, 1)
	tmpMap["liveStreamID"] = streamID
	jsonBytes, _ := json.Marshal(tmpMap)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Println(err)
		return false
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("isActive sent a request failed. %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
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

	isOnline = make(map[string]bool)
	log.Println("Starting server... Port is ", port)
	err = srv.ListenAndServe()
	if err != nil {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
