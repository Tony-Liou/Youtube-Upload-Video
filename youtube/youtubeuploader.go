package ytuploader

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// VideoSetting setting the video info that will be shown or be configured on Youtube
type VideoSetting struct {
	Filename    string // Filename is a filename
	Title       string
	Description string
	Category    string // default is 22
	Keywords    string // seperate by comma
	Privacy     string // public, unlisted, and private
	Language    string
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(scope string) *http.Client {
	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Printf("Unable to read client secret file: %v", err)
		return nil
	}

	// If modifying the scope, delete your previously saved credentials
	// at ~/.credentials/youtube-go.json
	config, err := google.ConfigFromJSON(b, scope)
	if err != nil {
		log.Printf("Unable to parse client secret file to config: %v", err)
		return nil
	}

	// Use a redirect URI like this for a web app. The redirect URI must be a
	// valid one for your OAuth2 credentials.
	config.RedirectURL = "http://localhost:8090"
	// Use the following redirect URI if launchWebServer=false in oauth2.go
	// config.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"

	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Printf("Unable to get path to cached credential file. %v", err)
		return nil
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		log.Println(err)
	}
	return config.Client(ctx, tok)
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("youtube-go.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

func checkVideoInfo(v *VideoSetting) {
	if v.Title == "" {
		v.Title = "Default title"
	}

	if v.Description == "" {
		v.Description = "Description"
	}

	if v.Category == "" {
		v.Category = "22"
	}

	if v.Privacy == "" {
		v.Privacy = "private"
	}

	if v.Language == "" {
		v.Language = "zh-TW"
	}
}

// UploadVideo will upload a video to Youtube.
// And you can use this function to approach this
func UploadVideo(v *VideoSetting) string {
	if v.Filename == "" {
		log.Println("You must provide a filename of a video file to upload")
		return ""
	}

	checkVideoInfo(v)

	client := getClient(youtube.YoutubeUploadScope)

	if client == nil {
		log.Println("Upload the video to Youtube failed")
		return ""
	}

	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Printf("Error creating YouTube client: %v", err)
		return ""
	}

	upload := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:                v.Title,
			Description:          v.Description,
			CategoryId:           v.Category,
			DefaultAudioLanguage: v.Language,
			DefaultLanguage:      v.Language,
		},
		Status: &youtube.VideoStatus{PrivacyStatus: v.Privacy},
	}

	// The API returns a 400 Bad Request response if tags is an empty string.
	if strings.Trim(v.Keywords, "") != "" {
		upload.Snippet.Tags = strings.Split(v.Keywords, ",")
	}

	call := service.Videos.Insert([]string{"snippet", "status"}, upload)

	file, err := os.Open(v.Filename)
	if err != nil {
		log.Printf("Error opening %v: %v", v.Filename, err)
		return ""
	}
	defer file.Close()

	log.Println("Starting uploading the video", v.Filename)
	response, err := call.Media(file).Do()
	if err != nil {
		log.Println(err)
		return ""
	}
	log.Println("Video uploaded")

	return response.Id
}
