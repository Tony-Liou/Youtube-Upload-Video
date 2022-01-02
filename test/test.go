package main

import (
	"fmt"
	"net/http"
	"os"

	ytuploader "github.com/Tony-Liou/Youtube-Upload-Video/youtube"
)

func redirectionHandler(w http.ResponseWriter, req *http.Request) {
	code := req.FormValue("code")
	if code != "" {
		w.Write([]byte("Copy the following auth code and paste it to the terminal:\n\n" + code))
	} else {
		w.Write([]byte("Error: " + req.FormValue("error")))
	}
}

func main() {
	srv := &http.Server{Addr: ":8090"}
	http.HandleFunc("/", redirectionHandler)
	go func() {
		err := srv.ListenAndServe()
		fmt.Println(err)
	}()

	// Only needed in first try
	fmt.Println("Creating OAuth 2 token file...")
	err := ytuploader.CreateOauthToken()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		fmt.Println("Create the oauth 2 token file at ~/.credentials successfully")
	}
	srv.Close()

	settings := &ytuploader.VideoSetting{
		Filename:    "sample.mp4",
		Title:       "Test video",
		Description: "Test description",
		Keywords:    "test,first try,github,tony-liou",
		Category:    "22",
		Privacy:     "private",
	}

	fmt.Println("Uploading...")
	videoId, err := ytuploader.UploadVideo(settings)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	} else {
		fmt.Println(videoId)
	}
}
