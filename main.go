package main

/*
TODO:
* Stage 2*
- Build UI for QNAP
- Integrate with QNAP API
- Release to QNAP
 */

import (
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"io/ioutil"
	"log"
	"path/filepath"
	"os"
	"net/url"
	"os/user"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"flag"
)

type Config struct {
	Storage string
	Filter string
}

// Command-line flags
var (
	directory = flag.String("data", "./", "Path to directory where you want the backup saved.")
	search = flag.String("filter", "", "A Drive search query used to filter what files you want" +
		" to backup.")
	configure = flag.Bool("configure", false, "Configure authentication tokens.")
)

var  (
	conf = Config{}
	dirs = make(map[string]string)
	err error
)

// Writes a downloaded file to the filesystem
func write_file(filename string, resp http.Response) {
	log.Printf("Writing file to %v", filename)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Not able to read file: %v", err)
	}
	ioutil.WriteFile(filename, b, 0755)
}

// Discover the full directory path for a file
func discover_dir_tree(i *drive.File, srv drive.Service) string {
	// If there are no parents use root
	if len(i.Parents) < 1 {
		return ""
	}

	// Checked for cached value
	if dir, ok := dirs[i.Parents[0]]; ok {
		return dir
	}

	// Find a result and cache it
	var last_file *drive.File
	last_file = i
	var parent_path string
	var err error
	for {
		if len(last_file.Parents) < 1 {
			break
		}

		last_file, err = srv.Files.Get(last_file.Parents[0]).Fields("id, name, parents").Do()
		if err != nil {
			log.Fatalf("Not able to get parent item: %v", err)
		}
		parent_path = fmt.Sprintf("%v/%v", last_file.Name, parent_path)
	}

	dirs[i.Parents[0]] = parent_path

	// Create folders
	os.MkdirAll(fmt.Sprintf("%v/%v", conf.Storage, parent_path), 0755)

	return parent_path
}

// Downloads a file from Google Drive
func download_file(i *drive.File, srv drive.Service) {
	if i.MimeType == "application/vnd.google-apps.folder" {
		log.Printf("Folder: %v\n", i.Name)
		return
	}

	// Get file's directory path
	file_dir := fmt.Sprintf("%v/%v", conf.Storage, discover_dir_tree(i, srv))
	log.Printf("Path: %v", file_dir)

	// Determine filename based on the type of file
	var filename string
	var download_type string
	if i.Size > 0 {
		filename = fmt.Sprintf("%v%v", file_dir, i.OriginalFilename)
		download_type = "binary"
	} else {
		log.Printf("Native type: %v\n", i.MimeType)

		switch i.MimeType {
		case "application/vnd.google-apps.spreadsheet":
			filename = fmt.Sprintf("%v%v.xlsx", file_dir, i.Name)
			download_type = "sheet"
		case "application/vnd.google-apps.document":
			filename = fmt.Sprintf("%v%v.docx", file_dir, i.Name)
			download_type = "doc"
		case "application/vnd.google-apps.drawing":
			filename = fmt.Sprintf("%v%v.pdf", file_dir, i.Name)
			download_type = "drawing"
		case "application/vnd.google-apps.presentation":
			filename = fmt.Sprintf("%v%v.pdf", file_dir, i.Name)
			download_type = "slides"
		case "application/vnd.google-apps.script":
			filename = fmt.Sprintf("%v%v.json", file_dir, i.Name)
			download_type = "script"
		default:
			log.Println("We don't have an export type for this file.")
			return
		}
	}

	// Check if file has been modified since the saved version
	file_stat, err := os.Stat(filename)
	if err == nil {
		mod_time, err := time.Parse(time.RFC3339, i.ModifiedTime)
		if err != nil {
			log.Fatalf("Not able to parse modifiedtime: %s", err)
			return
		}

		file_mod_time := file_stat.ModTime()
		log.Printf("File last modified: %v", file_mod_time)
		if mod_time.Before(file_mod_time) {
			log.Printf("Already latest: %v", filename)
			return
		}
	}

	// Download file
	log.Printf("Saving file: %s", i.Name)
	var resp *http.Response
	switch download_type {
	case "binary":
		log.Printf("Downloading binary file: %v\n", i.Name)
		file_get := srv.Files.Get(i.Id)
		resp, err = file_get.Download()
		if err != nil {
			log.Printf("Error downloading file: %v", err)
			return
		}
	case "sheet":
		log.Println("Exporting to MS Excel")
		export_call := srv.Files.Export(i.Id, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		resp, err = export_call.Download()
		if err != nil {
			log.Printf("Was not able to download file: %s\n", err)
			return
		}
	case "doc":
		log.Println("Exporting to MS Word")
		export_call := srv.Files.Export(i.Id, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		resp, err = export_call.Download()
		if err != nil {
			log.Printf("Was not able to download file: %s\n", err)
			return
		}
	case "drawing":
		log.Println("Exporting to PDF")
		export_call := srv.Files.Export(i.Id, "application/pdf")
		resp, err = export_call.Download()
		if err != nil {
			log.Printf("Was not able to download file: %s\n", err)
			return
		}
	case "slides":
		log.Println("Exporting to PDF")
		export_call := srv.Files.Export(i.Id, "application/pdf")
		resp, err = export_call.Download()
		if err != nil {
			log.Printf("Was not able to download file: %s\n", err)
			return
		}
	case "script":
		log.Println("Exporting to JSON")
		export_call := srv.Files.Export(i.Id, "application/vnd.google-apps.script+json")
		resp, err = export_call.Download()
		if err != nil {
			log.Printf("Was not able to download file: %s\n", err)
			return
		}
	default:
		log.Fatalf("No corresponding media type: %s", download_type)
		return
	}

	write_file(filename, *resp)
}

// Runs the backup process listing files from Google Drive
func run_backup(srv drive.Service) {
	pageToken := ""
	for {
		var r *drive.FileList
		list_call := srv.Files.List().PageToken(pageToken).PageSize(50).
			Fields("nextPageToken, files(id, name, mimeType, size, originalFilename, parents, modifiedTime)")
		if conf.Filter == "" {
			r, err = list_call.Do()
		} else {
			r, err = list_call.Q(conf.Filter).Do()
		}
		if err != nil {
			log.Fatalf("Unable to list files: %v", err)
		}

		if len(r.Files) > 0 {
			for _, i := range r.Files {
				download_file(i, srv)
			}
		} else {
			log.Print("End of file list")
		}

		pageToken = r.NextPageToken
		if pageToken == "" {
			log.Println("Finished download loop.")
			break
		}
	}
}

func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

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

func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("drive-backup-qnap.json")), err
}

func main() {
	ctx := context.Background()

	flag.Parse()
	conf.Storage = *directory
	conf.Filter = *search

	b := []byte("{\"installed\":{\"client_id\":\"509615771323-pntgak8thbl9ia0sd3gtb0t31utkkn6m.apps.googleusercontent.com\",\"project_id\":\"drive-backup-156821\",\"auth_uri\":\"https://accounts.google.com/o/oauth2/auth\",\"token_uri\":\"https://accounts.google.com/o/oauth2/token\",\"auth_provider_x509_cert_url\":\"https://www.googleapis.com/oauth2/v1/certs\",\"client_secret\":\"LvoP9X8RAZC3J6adRN3o-Xvb\",\"redirect_uris\":[\"urn:ietf:wg:oauth:2.0:oob\",\"http://localhost\"]}}")

	// Read-only scope is critical to prevent any changes/mistakes to the production Drive
	// We don't care about restore. User would manually restore files from the backup
	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client id file to config: %v", err)
	}

	client := getClient(ctx, config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to get Drive client: %v", err)
	}

	if !*configure {
		run_backup(*srv)
	} else {
		fmt.Println("Configuration complete!")
	}
}
