package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"
	middleware "gopkg.in/telebot.v4/middleware"
)

var (
	Version string
	Users   sync.Map
)

type Stats struct {
	TotalRequests  uint64 `json:"total_requests"`
	TempFilesCount int    `json:"temp_files_count"`
}

type User struct {
	Username  string    `json:"username"`
	IsBot     bool      `json:"is_bot"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	imageProcessingServerURL = getEnv("IMAGE_PROCESSING_SERVER_URL", "http://localhost:8080")
	imageProcessingAPIToken  = getEnv("IMAGE_PROCESSING_API_TOKEN", "")
	usersFilePath            = "users.json"
)

func main() {
	loadUsers()

	// Create a channel to receive signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	// Start a goroutine to handle the signal
	go func() {
		<-sigChan
		saveUsers()
		os.Exit(0)
	}()

	pref := tele.Settings{
		Token:  os.Getenv("TG_BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	if enableLogger := os.Getenv("TG_ENABLE_LOGGER"); enableLogger != "" {
		b.Use(middleware.Logger())
	}

	b.Handle("/hello", func(c tele.Context) error {
		user := c.Sender()
		if user != nil {
			welcomeMessage := fmt.Sprintf("Hello, %s! I can gray scale your photos!", user.FirstName)
			return c.Send(welcomeMessage)
		} else {
			return c.Send("Hello, I can gray scale your photos!")
		}
	})

	b.Handle("/help", func(c tele.Context) error {
		return c.Send("Upload your photo and we'll gray scale it for you!")
	})

	b.Handle("/settings", func(c tele.Context) error {
		return c.Send("Settings not implemented yet!")
	})

	b.Handle("/start", func(c tele.Context) error {
		user := c.Sender()
		if user != nil {
			welcomeMessage := fmt.Sprintf("Welcome, %s!", user.FirstName)
			storeUser(user)
			return c.Send(welcomeMessage)
		} else {
			return c.Send("Welcome!")
		}
	})

	b.Handle("/stats", func(c tele.Context) error {
		stats, err := getStats()
		if err != nil {
			return c.Send("Failed to retrieve stats: " + err.Error())
		}
		return c.Send(fmt.Sprintf("Stats: %d reqs, %d files", stats.TotalRequests, stats.TempFilesCount))
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		// Handle all text messages that are not commands here, if needed.
		// If you don't need special handling, you can leave it empty or just have a default response.
		return c.Send("You said: " + c.Text())
	})

	b.Handle(tele.OnPhoto, func(c tele.Context) error {
		photo := c.Message().Photo
		file := &tele.File{FileID: photo.FileID}

		// Create a temporary file to save the downloaded photo
		tempFilePath := filepath.Join(os.TempDir(), photo.FileID+".jpg")
		if err := b.Download(file, tempFilePath); err != nil {
			return c.Send("Failed to download photo.")
		}

		// Inform the user that their photo is being processed
		if err := c.Send("Your photo is being processed. Please wait..."); err != nil {
			return err
		}

		// Send the photo to the image processing server
		processedImage, err := processImage(tempFilePath)
		if err != nil {
			return c.Send("Failed to process image: " + err.Error())
		}

		// Send the processed image back to the user
		return c.Send(&tele.Photo{File: tele.FromReader(processedImage)})
	})

	log.Println("Bot started.")
	defer saveUsers()
	b.Start()
}

func storeUser(user *tele.User) {
	if _, exists := Users.Load(user.ID); !exists {
		newUser := User{
			Username:  user.Username,
			IsBot:     user.IsBot,
			CreatedAt: time.Now(),
		}
		Users.Store(user.ID, newUser)
	}
}

func loadUsers() {
	file, err := os.Open(usersFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatalf("Failed to open users file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	users := make(map[int64]User)
	if err := decoder.Decode(&users); err != nil {
		log.Fatalf("Failed to decode users file: %v", err)
	}

	for id, user := range users {
		Users.Store(id, user)
	}
	log.Println("Users loaded from file.")
}

func saveUsers() {
	file, err := os.Create(usersFilePath)
	if err != nil {
		log.Fatalf("Failed to create users file: %v", err)
	}
	defer file.Close()

	users := make(map[int64]User)
	Users.Range(func(key, value interface{}) bool {
		id, ok := key.(int64)
		if !ok {
			return false
		}
		user, ok := value.(User)
		if !ok {
			return false
		}
		users[id] = user
		return true
	})

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(users); err != nil {
		log.Fatalf("Failed to encode users to file: %v", err)
	}
	log.Println("Users saved to file.")
}

func getStats() (Stats, error) {
	resp, err := http.Get(fmt.Sprintf("%s/stats", imageProcessingServerURL))
	result := Stats{}
	if err != nil {
		return result, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("server returned non-200 status: %v", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("failed to decode response: %v", err)
	}

	return result, nil
}

func processImage(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create a buffer to store the multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a form field for the file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	// Copy the file content to the form field
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %v", err)
	}

	// Close the writer to finalize the form
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %v", err)
	}

	// Create a new HTTP request to send the form data
	req, err := http.NewRequest("POST", imageProcessingServerURL+"/image", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add the API token to the request header if it's set
	if imageProcessingAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+imageProcessingAPIToken)
	}

	// Send the request to the server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-200 status: %v", resp.Status)
	}

	// Parse the response to get the image ID
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Wait for 1 second before requesting the processed image
	time.Sleep(1 * time.Second)

	// Request the processed image using the image ID
	imageURL := fmt.Sprintf("%s/image?id=%s", imageProcessingServerURL, result.ID)
	req, err = http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	if imageProcessingAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+imageProcessingAPIToken)
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-200 status: %v", resp.Status)
	}

	// Read the response body into a buffer
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return &buf, nil
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
