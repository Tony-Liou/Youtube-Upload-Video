package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/Tony-Liou/Youtube-Upload-Video/myUpload"
)

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

func notifyVideoId(videoID string) {
	url := os.Getenv("notification-url")
	if url == "" {
		return
	}

	url += "?videoID=" + videoID

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
	url := "https://api-dsa.17app.co/api/v1/lives/" + streamID + "/viewers/alive"

	tmpMap := map[string]string{"liveStreamID": streamID}
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

	notifyVideoId(videoID)
}
