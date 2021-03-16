# Youtube Upload Video

Download a [17 Live](https://17.live/) live streaming and upload the dumped video to Youtube after the stream is ended.

## Installation

### Environment

- Linux (Ubuntu 16.04 or later)

### Prerequisite

- [Golang](https://golang.org/dl/) (1.16 or later)
- [Streamlink](https://github.com/streamlink/streamlink/releases/latest)
  - [RTMPDump](http://rtmpdump.mplayerhq.hu/) (if Streamlink did not installed automatically)
- [Youtube Data API](https://developers.google.com/youtube/v3/getting-started#before-you-start) (enable it and get the `client_secret.json`)

## Usage

1. Clone this repo
2. Delete some functions in `streaminghandler.go` that you do not want to use.
   E.g.,
   ```go
   func notifyVideoId(videoID string)
   ```
3. Build and run `HttpServer.go` then it will work
   ```shell
   go run HttpServer.go streaminghandler.go
   ```
4. Press `Ctrl+C` to terminate this app if you want to stop it

## Credit

My Github web page template is powered by [HTML5 UP](https://html5up.net/)

## License

MIT