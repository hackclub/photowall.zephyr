package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed index.tmpl.html
var indexTmplRaw string
var indexTmpl *template.Template

func main() {
	// Support for auto-deploy .zephyr stuff
	port := os.Getenv("PORT")
	if port == "" {
		port = "9876"
	}

	var err error
	indexTmpl, err = template.New("index.html").Parse(indexTmplRaw)
	if err != nil {
		log.Fatal(err)
	}

	// HTTP Request handlers
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/photos/", photoHandler)
	http.HandleFunc("/static/", staticHandler)
	http.HandleFunc("/upload", uploadHandler)

	fmt.Println("Listening on port", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	files, err := ioutil.ReadDir("./db/")
	if err != nil {
		fmt.Fprintln(os.Stderr, "request to /, error:", err)
		http.Error(w, "error getting photos from db", 500)
		return
	}

	photoURLs := []string{}
	for _, file := range files {
		if file.Name() == "README.md" || file.Name() == ".gitignore" {
			continue
		}

		photoURLs = append(photoURLs, "/photos/"+file.Name())
	}

	sort.Strings(photoURLs)
	sort.Sort(sort.Reverse(sort.StringSlice(photoURLs)))

	data := struct {
		Photos []string
	}{
		Photos: photoURLs,
	}

	if err := indexTmpl.Execute(w, data); err != nil {
		fmt.Fprintln(os.Stderr, "error executing index.html template:", err)
	}
}

func photoHandler(w http.ResponseWriter, req *http.Request) {
	fileName := strings.Replace(req.URL.Path, "/photos/", "", 1)

	file, err := os.Open("./db/" + fileName)
	if err != nil {
		http.Error(w, "error opening photo: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	defer file.Close()

	io.Copy(w, file)
}

func staticHandler(w http.ResponseWriter, req *http.Request) {
	fileName := strings.Replace(req.URL.Path, "/static/", "", 1)

	file, err := os.Open("./static/" + fileName)
	if err != nil {
		http.Error(w, "error opening photo: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "text/css")
	io.Copy(w, file)
}

// Handler for uploads
func uploadHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("upload endpoint hit by", req.RemoteAddr)

	req.ParseMultipartForm(10 << 20) // 10mb file limit

	// Recieved data from the html upload form
	photo, handler, err := req.FormFile("photo")
	if err != nil {
		http.Error(w, "error receiving uploaded photo: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	defer photo.Close()

	// Verify that it is, in fact, a photo
	ext := strings.ToLower(filepath.Ext(handler.Filename))

	if ext != ".png" && ext != ".jpeg" && ext != ".jpg" {
		http.Error(w, "must be a png or jpeg", http.StatusUnprocessableEntity)
		return
	}

	unixTime := strconv.FormatInt(time.Now().Unix(), 10)

	file, err := ioutil.TempFile("db/", "upload-"+unixTime+"-*"+ext)
	if err != nil {
		http.Error(w, "internal error storing file: "+err.Error(), 500)
		return
	}
	defer file.Close()

	io.Copy(file, photo)

	http.Redirect(w, req, "/", http.StatusFound)
}
