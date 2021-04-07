package gdrive

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "gdrive_token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// getOrCreateFolder returns the folder Id
func getOrCreateFolder(d *drive.Service, folderName string) string {
	if folderName == "" {
		return ""
	}
	q := fmt.Sprintf(`name="%s" and mimeType="application/vnd.google-apps.folder"`, folderName)

	r, err := d.Files.List().Q(q).PageSize(1).Do()
	if err != nil {
		log.Fatalln("Unable to retrieve foldername.", err)
	}

	folderId := ""
	if len(r.Files) > 0 {
		folderId = r.Files[0].Id
	} else {
		fmt.Printf("Folder not found. Create new folder: %s\n", folderName)
		f := &drive.File{Name: folderName, Description: "Auto create by gdrive-upload", MimeType: "application/vnd.google-apps.folder"}
		r, err := d.Files.Create(f).Do()
		if err != nil {
			fmt.Printf("An error occurred when create folder: %v\n", err)
		}
		folderId = r.Id
	}
	return folderId
}

func createFile(parentId, name, desc, cTime, mimeType string) *drive.File {
	return &drive.File{
		MimeType:    mimeType,
		Name:        name,
		Description: desc,
		Parents:     []string{parentId},
		CreatedTime: cTime,
	}
}

func uploadFile(srv *drive.Service, fileInfo *drive.File, content io.Reader) (*drive.File, error) {
	file, err := srv.Files.Create(fileInfo).Media(content).Do()
	if err != nil {
		return nil, err
	}

	return file, nil
}

// UploadVideo uploads the file to the target Google Drive
func UploadVideo(filePath, driveFileName, mimeType string) {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	f, err := os.Open(filePath)
	if err != nil {
		log.Printf("Cannot open the file: %v\n", err)
		return
	}

	folderId := getOrCreateFolder(srv, "Streaming backup")
	fileInfo := createFile(folderId, driveFileName, "", "", mimeType)
	file, err := uploadFile(srv, fileInfo, f)
	if err != nil {
		log.Fatalf("Could not create drive file: %v", err)
	}

	log.Printf("%s uploaded to Google Drive.\n", file.Name)
}
