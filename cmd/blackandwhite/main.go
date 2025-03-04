package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

type ImageMetadata struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata"`
}

type ImageResponse struct {
	ID string `json:"id"`
}

type Stats struct {
	TotalRequests  uint64 `json:"total_requests"`
	TempFilesCount int    `json:"temp_files_count"`
}

var tempDir = getEnv("TEMP_DIR", "tmp")
var apiToken = getEnv("API_TOKEN", "")
var requestCounter uint64

func main() {
	// Ensure the temporary directory exists
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create temporary directory: %v\n", err)
	}

	// Start the file cleanup goroutine
	go cleanupOldFiles()

	http.HandleFunc("POST /image", authMiddleware(imageHandler))
	http.HandleFunc("GET /image", authMiddleware(getImageHandler))
	http.HandleFunc("GET /stats", statsHandler)

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" || token != "Bearer "+apiToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		atomic.AddUint64(&requestCounter, 1)
		next.ServeHTTP(w, r)
	}
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Retrieve the file from the form data
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate a unique ID for the image
	id := uuid.New().String()
	tempFilePath := filepath.Join(tempDir, id+".jpg")

	// Create a temporary file to save the uploaded image
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	// Convert the image to black and white and save it
	if err := convertToBlackAndWhite(file, tempFile); err != nil {
		http.Error(w, "Failed to convert image", http.StatusInternalServerError)
		return
	}

	// Respond with the image ID
	response := ImageResponse{ID: id}
	jsonResponse(w, response)
}

func getImageHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	imageID := sanitizeUUIDQueryParam(queryParams, "id")

	// Re-associate the metadata with the image
	tempFilePath := filepath.Join(tempDir, imageID+".jpg")
	if _, err := os.Stat(tempFilePath); os.IsNotExist(err) {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// Open the image file
	imgFile, err := os.Open(tempFilePath)
	if err != nil {
		http.Error(w, "Failed to open image file", http.StatusInternalServerError)
		return
	}
	defer imgFile.Close()

	// Decode the image
	img, _, err := image.Decode(imgFile)
	if err != nil {
		http.Error(w, "Failed to decode image", http.StatusInternalServerError)
		return
	}

	// Respond with the image
	w.Header().Set("Content-Type", "image/jpeg")
	if err := jpeg.Encode(w, img, nil); err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		return
	}
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	numFiles, _ := getNumberOfFiles(tempDir)
	stats := &Stats{
		TotalRequests:  atomic.LoadUint64(&requestCounter),
		TempFilesCount: numFiles,
	}
	jsonResponse(w, stats)
}

func convertToBlackAndWhite(input io.Reader, output io.Writer) error {
	// Decode the image
	img, _, err := image.Decode(input)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	// Convert the image to grayscale
	grayImg := imaging.Grayscale(img)

	// Encode and save the resulting image
	err = jpeg.Encode(output, grayImg, nil)
	if err != nil {
		return fmt.Errorf("failed to encode and save image: %v", err)
	}

	return nil
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func sanitizeUUIDQueryParam(query url.Values, paramName string) string {
	value := query.Get(paramName)
	if value == "" {
		return "" // Or handle missing parameter as needed
	}

	// Allowed UUID characters: alphanumeric and hyphen
	re := regexp.MustCompile("[^a-fA-F0-9-]")
	sanitized := re.ReplaceAllString(value, "")

	return sanitized
}

// getNumberOfFiles returns the number of files in the given directory
func getNumberOfFiles(dirPath string) (int, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %v", err)
	}
	return len(files), nil
}

func cleanupOldFiles() {
	for {
		time.Sleep(1 * time.Hour)
		files, err := os.ReadDir(tempDir)
		if err != nil {
			log.Printf("Failed to read temp directory: %v\n", err)
			continue
		}

		cutoff := time.Now().Add(-24 * time.Hour)
		for _, file := range files {
			filePath := filepath.Join(tempDir, file.Name())
			info, err := os.Stat(filePath)
			if err != nil {
				log.Printf("Failed to stat file: %v\n", err)
				continue
			}
			if info.ModTime().Before(cutoff) {
				if err := os.Remove(filePath); err != nil {
					log.Printf("Failed to remove file: %v\n", err)
				} else {
					log.Printf("Removed old file: %s\n", filePath)
				}
			}
		}
	}
}
