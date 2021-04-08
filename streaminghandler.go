package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	gdrive "github.com/Tony-Liou/Youtube-Upload-Video/google-drive"
	ytuploader "github.com/Tony-Liou/Youtube-Upload-Video/youtube"
)

// Dump the target stream, return a current time string and a filename (with path)
func execStreamlink(streamURL string) (string, string) {
	t := time.Now()
	curTime := t.Format("2006/01/02_15:04:05_Mon")
	curTimeStamp := strconv.FormatInt(t.UnixNano(), 10)

	app := "streamlink"

	option := "-o"
	filename := "stream" + curTimeStamp
	url := streamURL
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
	url := os.Getenv("NOTIFICATION_URL")
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

// Executing streamlink to dump the live streaming.
// After the live streaming ending, upload the video to Youtube.
func processStreaming(streamURL, privacy string) {
	log.Println("Processing streaming...")

	isDownloading = true
	recordTime, uri := execStreamlink(streamURL)
	isDownloading = false

	setting := &ytuploader.VideoSetting{
		Filename:    uri,
		Title:       recordTime,
		Description: streamURL,
		Category:    "22",
		Privacy:     privacy,
	}
	videoID := ytuploader.UploadVideo(setting)
	if videoID == "" {
		log.Println("Upload video failed. Starting uploading video to Gogole Drive.")
		gdrive.UploadVideo(uri, recordTime, "")
	} else {
		log.Println("Video uploaded. ID: ", videoID)

		removeVideoFile(uri)

		notifyVideoId(videoID)
	}
}
