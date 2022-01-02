package main

import (
	"bytes"
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
		log.Printf("%v : %s\n", err, stderr.String())
	}
	log.Println("Result: ", out.String())

	return curTime, filename
}

func removeVideoFile(path string) {
	err := os.Remove(path)
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

// processStreaming Executes streamlink to dump the live streaming.
// After the live streaming ending, upload the video to YouTube.
func processStreaming(streamURL, privacy string) {
	log.Println("Processing streaming...")

	recordTime, uri := execStreamlink(streamURL)

	setting := &ytuploader.VideoSetting{
		Filename:    uri,
		Title:       recordTime,
		Description: streamURL,
		Category:    "22",
		Privacy:     privacy,
		//Language:    "zh-TW",
	}
	videoID, err := ytuploader.UploadVideo(setting)
	if err != nil {
		log.Println("Upload video failed. Starting uploading video to Google Drive.")
		gdrive.UploadVideo(uri, recordTime, "")
	} else {
		log.Println("Video uploaded. ID: ", videoID)

		removeVideoFile(uri)

		notifyVideoId(videoID)
	}
}
