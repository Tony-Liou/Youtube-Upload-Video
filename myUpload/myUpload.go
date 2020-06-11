package myUpload

import (
	"context"
	"encoding/json"
	"fmt"
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

const missingClientSecretsMessage = `
Please configure OAuth 2.0
To make this sample run, you need to populate the client_secrets.json file
found at:
   %v
with information from the {{ Google Cloud Console }}{{ https://cloud.google.com/console }}For more information about the client_secrets.json file format, please visit:
https://developers.google.com/api-client-library/python/guide/aaa_client_secrets
`

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
		authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

		fmt.Println("Trying to get token from prompt")
		tok, err = getTokenFromPrompt(config, authURL)

		if err == nil {
			saveToken(cacheFile, tok)
		}
	}
	return config.Client(ctx, tok)
}

// Exchange the authorization code for an access token
func exchangeToken(config *oauth2.Config, code string) (*oauth2.Token, error) {
	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Printf("Unable to retrieve token %v", err)
		return nil, err
	}
	return tok, nil
}

// getTokenFromPrompt uses Config to request a Token and prompts the user
// to enter the token on the command line. It returns the retrieved Token.
func getTokenFromPrompt(config *oauth2.Config, authURL string) (*oauth2.Token, error) {
	var code string
	fmt.Printf("Go to the following link in your browser. After completing "+
		"the authorization flow, enter the authorization code on the command "+
		"line: \n%v\n", authURL)

	if _, err := fmt.Scan(&code); err != nil {
		log.Printf("Unable to read authorization code %v", err)
		return nil, err
	}
	fmt.Println(authURL)
	return exchangeToken(config, code)
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
func saveToken(file string, token *oauth2.Token) {
	fmt.Println("trying to save token")
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Printf("Unable to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// VideoSetting is a struct
type VideoSetting struct {
	Filename    string // Filename is a filename
	Title       string
	Description string
	Category    string // default is 22
	Keywords    string // seperate by comma
	Privacy     string // public, unlisted, and private
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

// UploadVideo will upload a video to Youtube.
// And you can use this function to approach this
func UploadVideo(v *VideoSetting) string {

	checkVideoInfo(v)

	if v.Filename == "" {
		log.Printf("You must provide a filename of a video file to upload")
		return ""
	}

	client := getClient(youtube.YoutubeUploadScope)

	if client == nil {
		log.Println("Upload failed")
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
			Title:       v.Title,
			Description: v.Description,
			CategoryId:  v.Category,
		},
		Status: &youtube.VideoStatus{PrivacyStatus: v.Privacy},
	}

	// The API returns a 400 Bad Request response if tags is an empty string.
	if strings.Trim(v.Keywords, "") != "" {
		upload.Snippet.Tags = strings.Split(v.Keywords, ",")
	}

	call := service.Videos.Insert("snippet,status", upload)

	file, err := os.Open(v.Filename)
	defer file.Close()
	if err != nil {
		log.Printf("Error opening %v: %v", v.Filename, err)
		return ""
	}

	response, err := call.Media(file).Do()
	if err != nil {
		log.Println(err)
		return ""
	}

	fmt.Printf("Upload successful! Video ID: %v\n", response.Id)
	return response.Id
}
