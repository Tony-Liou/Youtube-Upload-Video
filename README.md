# Youtube Upload Video

Download a live-streaming and upload the dumped video to YouTube after the stream is ended.
If upload video to YouTube failed, uploading to Google Drive instead.

## Installation

### Environment

- Linux (Tested on Ubuntu 20.04)

### Prerequisites

- [Golang](https://golang.org/dl/) (1.16 or later)
- [Streamlink](https://github.com/streamlink/streamlink/releases/latest)
  - [RTMPDump](http://rtmpdump.mplayerhq.hu/) (if Streamlink did not install it automatically)
- [Youtube Data API](https://developers.google.com/youtube/v3/getting-started#before-you-start) (enable it and get the `client_secret.json`)
- [Google Drive API](https://developers.google.com/drive/api/v3/enable-drive-api#enable_the_drive_api) (Optional)

## Usage

1. Clone this repo
2. Delete some functions in `streaminghandler.go` that you do not want to use.
   E.g.,
   ```go
   func notifyVideoId(videoID string)
   ```
3. Build and run `HttpServer.go` then it will be ready to go
   ```shell
   go run HttpServer.go streaminghandler.go
   ```
4. Press `Ctrl+C` to stop it

## Credit

My GitHub web page template is powered by [HTML5 UP](https://html5up.net/)

## License

MIT