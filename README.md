# Youtube Upload Video

[![Codacy Badge](https://api.codacy.com/project/badge/Grade/d5037560580c4046977760c7bb73cd76)](https://app.codacy.com/manual/Tony-Liou/Youtube-Upload-Video?utm_source=github.com&utm_medium=referral&utm_content=Tony-Liou/Youtube-Upload-Video&utm_campaign=Badge_Grade_Dashboard)

Download a [17 Live](https://17.live/) live streaming and upload the dumped video to Youtube after the stream is ended.

## Installation

### Environment

- Linux (Ubuntu 16.04 or later)

### Prerequisite

- [Golang](https://golang.org/dl/) (1.11 or later)
- [Streamlink](https://github.com/streamlink/streamlink/releases/latest)
  + [RTMPDump](http://rtmpdump.mplayerhq.hu/) (if Streamlink did not installed automatically)
- [Youtube Data API](https://developers.google.com/youtube/v3/getting-started#before-you-start) (enable it and get the `client_secret.json`)

## Usage

1. Clone this repo
2. Delete some functions in `HttpServer.go` that you do not want to use.
   E.g.,
   ```go
   func sendVideoInfo(videoID string)
   ```
3. Create a go module file in the top level directory of this project
   ```shell=
   go mod init myUpload
   go mod tidy
   ```
4. Build and run `HttpServer.go` then it will work
5. Press `Ctrl+C` to terminate this app if you want to stop it

## Credit

My Github web page template is powered by [HTML5 UP](https://html5up.net/)

## License

MIT