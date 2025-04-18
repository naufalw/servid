package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

	masterPlaylistPath := filepath.Join(videoHlsPath, "master.m3u8")

	variants := []struct {
		w, h  string // width, height
		vbit  string // video bitrate
		abits string // audio bitrate
	}{
		{"640", "360", "800k", "96k"},
		{"842", "480", "1400k", "128k"},
		{"1280", "720", "2800k", "128k"},
		{"1920", "1080", "5000k", "192k"},
	}

	// Make dir for the variants
	for i := range variants {
		dir := filepath.Join(videoHlsPath, fmt.Sprintf("v%d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			http.Error(w, "mkdir "+dir+": "+err.Error(), 500)
			return
		}
	}

	// build filter
	// split=4[v0in][v1in][v2in][v3in];
	// [v0in]scale=640:360[v0out];
	// [v1in]scale=842:480[v1out];
	splitCount := len(variants)
	inLabels := make([]string, splitCount)
	outLabels := make([]string, splitCount)
	for i := range splitCount {
		inLabels[i] = fmt.Sprintf("[v%din]", i)
		outLabels[i] = fmt.Sprintf("[v%dout]", i)
	}
	fcParts := []string{
		fmt.Sprintf("[0:v]split=%d%s", splitCount, strings.Join(inLabels, "")),
	}
	for i, v := range variants {
		fcParts = append(fcParts,
			fmt.Sprintf("%sscale=%s:%s%s", inLabels[i], v.w, v.h, outLabels[i]),
		)
	}
	filterComplex := strings.Join(fcParts, ";")

	// Base flags
	ffmpegArgs := []string{
		"-hide_banner", "-y",
		"-i", rawFilePath,
		"-filter_complex", filterComplex}

	// Map each scaled video + a copy of same audio
	for i := range variants {
		// video
		ffmpegArgs = append(ffmpegArgs,
			"-map", outLabels[i],
			"-b:v:"+strconv.Itoa(i), variants[i].vbit,
			"-c:v:"+strconv.Itoa(i), "libx264",
			// audio
			"-map", "0:a:0",
			"-b:a:"+strconv.Itoa(i), variants[i].abits,
			"-c:a:"+strconv.Itoa(i), "aac",
		)
	}

	// muxxxx hls
	segmentPattern :=
		filepath.Join(videoHlsPath, "v%v", "segment%03d.ts")

	streamMapLabels := make([]string, splitCount)
	for i := 0; i < splitCount; i++ {
		streamMapLabels[i] = fmt.Sprintf("v:%d,a:%d", i, i)
	}

	variantPlaylistPattern := filepath.Join(
		videoHlsPath,
		"v%v",           // substitute variant index (0,1,2…)
		"playlist.m3u8", // constant name inside each vX/
	)

	ffmpegArgs = append(ffmpegArgs,
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_list_size", "0",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", segmentPattern,
		"-master_pl_name", filepath.Base(masterPlaylistPath),
		"-var_stream_map", strings.Join(streamMapLabels, " "),
		variantPlaylistPattern, // writes the playlist.m3u8
	)

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	log.Println("ffmpeg args:", cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Println("ffmpeg failed:", err)
		http.Error(w, "encoding failed", 500)
		return
	}

	log.Printf("ffmpeg success for %s", videoID)

	w.WriteHeader(http.StatusOK)
	streamUrl := fmt.Sprintf("/stream/%s/master.m3u8", videoID)
	fmt.Fprintf(w, "File uploaded and encoded. Video ID: %s, Stream URL: %s", videoID, streamUrl)

}
