package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const uploadDir = "./uploads"

var fileMap = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

func main() {

	os.MkdirAll(uploadDir, os.ModePerm)

	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/files/", fileHandler)

	fmt.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filepath := filepath.Join(uploadDir, handler.Filename)
	out, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	fileMap.Lock()
	fileLink := fmt.Sprintf("/files/%s", handler.Filename)
	fileMap.m[handler.Filename] = fileLink
	fileMap.Unlock()

	fmt.Fprintf(w, "File uploaded successfully! Access it at: %sn", fileLink)
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/files/"):]

	fileMap.RLock()
	_, exists := fileMap.m[filename]
	fileMap.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filepath.Join(uploadDir, filename))
}
