package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	uploadPath    = "./uploads/raw"
	hlsPath       = "./uploads/hls"
	maxUploadSize = 500 * 1024 * 1024 // 500 MB Limit
)

func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	os.MkdirAll(uploadPath, os.ModePerm) // This is 777 permission
	os.MkdirAll(hlsPath, os.ModePerm)

	http.Handle("/ping", enableCors(http.HandlerFunc(pingHandler)))
	http.Handle("/upload", enableCors(http.HandlerFunc(uploadHandler)))

	hlsFileServer := http.FileServer(http.Dir(hlsPath))
	http.Handle("/stream/", http.StripPrefix("/stream/", enableCors(hlsFileServer)))

	port := "8080"
	fmt.Printf("Starting server on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only post allowed bro", http.StatusMethodNotAllowed)
		return
	}

	// This is to check 500 NB
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "Too big", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("video") // form field should be named video
	if err != nil {
		http.Error(w, "Cannot get the file", http.StatusBadRequest)
		return
	}

	defer file.Close()

	videoID := uuid.New().String()

	ext := filepath.Ext(fileHeader.Filename)
	rawFileName := videoID + ext
	rawFilePath := filepath.Join(uploadPath, rawFileName)

	dst, err := os.Create(rawFilePath)
	if err != nil {
		http.Error(w, "Cannot create file in server", http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}

	log.Printf("Uploaded file %s saved as %s\n", fileHeader.Filename, rawFilePath)

	videoHlsPath := filepath.Join(hlsPath, videoID)

	if err := os.MkdirAll(videoHlsPath, os.ModePerm); err != nil {
		http.Error(w, "Cannot create hls dir", http.StatusInternalServerError)
		return
	}

	outputPlaylist := filepath.Join(videoHlsPath, "playlist.m3u8")
	segmentFilename := filepath.Join(videoHlsPath, "segment%03d.ts")

	cmdArgs := []string{
		"-i", rawFilePath, //Input
		"-profile:v", "baseline",
		"-level", "3.0",
		"-start_number", "0",
		"-hls_time", "10",
		"-hls_list_size", "0",
		"-f", "hls", // HLS format
		"-hls_segment_filename", segmentFilename,
		outputPlaylist, // the m3u8 go here
	}

	cmd := exec.Command("ffmpeg", cmdArgs...)

	// Pipe ffmpeg to server logs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Starting ffmpeg encoding for %s", videoID)
	err = cmd.Run()

	if err != nil {
		log.Printf("ffmpeg error for %s %v", videoID, err)
		http.Error(w, "Encoding fail", http.StatusInternalServerError)
		return
	}

	log.Printf("ffmpeg success for %s", videoID)

	w.WriteHeader(http.StatusOK)
	streamUrl := fmt.Sprintf("/stream/%s/playlist.m3u8", videoID)
	fmt.Fprintf(w, "File uploaded and encoded. Video ID: %s, Stream URL: %s", videoID, streamUrl)

}
