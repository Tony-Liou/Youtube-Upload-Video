package ytuploader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// VideoSetting setting the video info that will be shown or be configured on YouTube
type VideoSetting struct {
	Filename    string
	Title       string
	Description string

	// default is 22
	Category string

	// separate by comma
	Keywords string

	// public, unlisted, and private
	Privacy  string
	Language string
}

// CreateOauthToken requests oauth token and saves it to a file.
func CreateOauthToken() error {
	b, err := os.ReadFile("client_secret.json")
	if err != nil {
		return err
	}

	// If modifying the scope, delete your previously saved credentials
	// at ~/.credentials/youtube-go.json
	config, err := google.ConfigFromJSON(b, youtube.YoutubeUploadScope)
	if err != nil {
		return err
	}

	// Use a redirect URI like this for a web app. The redirect URI must be a
	// valid one for your OAuth2 credentials.
	config.RedirectURL = "http://localhost:8090"

	tok := getTokenFromWeb(config)
	cacheFile, err := tokenCacheFile()
	if err != nil {
		return err
	}

	saveToken(cacheFile, tok)
	return nil
}

// UploadVideo uploads a video to YouTube according to VideoSetting.
func UploadVideo(v *VideoSetting) (string, error) {
	if v.Filename == "" {
		return "", errors.New("file name is empty")
	}

	checkVideoInfo(v)

	client, err := getClient(youtube.YoutubeUploadScope)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", err
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
		return "", err
	}
	defer file.Close()

	response, err := call.Media(file).Do()
	if err != nil {
		return "", err
	}

	return response.Id, nil
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
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(scope string) (*http.Client, error) {
	ctx := context.Background()

	b, err := os.ReadFile("client_secret.json")
	if err != nil {
		return nil, err
	}

	// If modifying the scope, delete your previously saved credentials
	// at ~/.credentials/youtube-go.json
	config, err := google.ConfigFromJSON(b, scope)
	if err != nil {
		return nil, err
	}

	// Use a redirect URI like this for a web app. The redirect URI must be a
	// valid one for your OAuth2 credentials.
	config.RedirectURL = "http://localhost:8090"
	// Use the following redirect URI if launchWebServer=false in oauth2.go
	// config.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"

	cacheFile, err := tokenCacheFile()
	if err != nil {
		return nil, err
	}
	tok, err := tokenFromFile(cacheFile)

	return config.Client(ctx, tok), err
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
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

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) (string, error) {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(token)

	return fmt.Sprintf("Saving credential file to: %s", file), err
}
