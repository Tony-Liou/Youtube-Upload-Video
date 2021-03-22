package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var isDownloading bool       // Streamlink is dumping the stream
var isOnline map[string]bool // key is the stream url and value indicates it is online or not

type frame struct {
	IsStreaming bool   `json:"isStreaming"`
	StreamURL   string `json:"streamURL"`
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

		if err = json.Unmarshal(reqBody, &ih.frame); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Println(err)
			return
		}

		if ih.frame.StreamURL != "" {
			isOnline[ih.frame.StreamURL] = ih.frame.IsStreaming
			if isOnline[ih.frame.StreamURL] && !isDownloading {
				go processStreaming(ih.frame.StreamURL)
			}
		}

		msg := fmt.Sprintf("%v is streaming: %v\n", ih.frame.StreamURL, ih.frame.IsStreaming)
		w.Write([]byte(msg))
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

func main() {
	logFileName := os.Getenv("LOG_FILE_NAME")
	if logFileName == "" {
		logFileName = "myServer.log"
	}
	f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
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
