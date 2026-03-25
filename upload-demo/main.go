package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	MaxFileSize     = 5 * 1024 * 1024 // 5MB
	UploadDirectory = "/var/lib/mediagate/uploads"
)

var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

var magicBytes = map[string][]byte{
	"image/jpeg": {0xFF, 0xD8, 0xFF},
	"image/png":  {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
	"image/gif":  {0x47, 0x49, 0x46, 0x38},
	"image/webp": {0x52, 0x49, 0x46, 0x46},
}

func main() {
	log.Println("Starting MediaGate server on :8080...")

	// Ensure upload directory exists with proper permissions
	if err := os.MkdirAll(UploadDirectory, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("POST /upload", uploadHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>MediaGate - Profile Image Upload</title>
</head>
<body>
    <h1>MediaGate Profile Image Service</h1>
    <p>Accepted formats: JPEG, PNG, GIF, WebP (Max size: 5MB)</p>
    <form action="/upload" method="post" enctype="multipart/form-data">
        <label for="profile_image">Choose profile image:</label><br>
        <input type="file" id="profile_image" name="profile_image" accept="image/*"><br><br>
        <input type="submit" value="Upload Image">
    </form>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxFileSize)

	// Parse multipart form with size limit
	err := r.ParseMultipartForm(MaxFileSize)
	if err != nil {
		log.Printf("Failed to parse form: %v", err)
		http.Error(w, "File too large or invalid form data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("profile_image")
	if err != nil {
		log.Printf("No file uploaded: %v", err)
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read first 512 bytes for content detection
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		log.Printf("Failed to read file header: %v", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Reset file pointer
	file.Seek(0, 0)

	// Detect content type using magic bytes
	contentType := http.DetectContentType(buffer)

	// Validate content type
	extension, allowed := allowedMimeTypes[contentType]
	if !allowed {
		log.Printf("Invalid file type: %s", contentType)
		http.Error(w, "Invalid file type. Only JPEG, PNG, GIF, and WebP images are allowed", http.StatusBadRequest)
		return
	}

	// Verify magic bytes match content type
	if !verifyMagicBytes(buffer, contentType) {
		log.Printf("Magic bytes don't match content type: %s", contentType)
		http.Error(w, "File content doesn't match file type", http.StatusBadRequest)
		return
	}

	// Validate file extension from original filename
	originalExt := strings.ToLower(filepath.Ext(header.Filename))
	expectedExt := extension
	if originalExt != expectedExt && !isValidExtensionForType(originalExt, contentType) {
		log.Printf("File extension mismatch: %s vs %s", originalExt, expectedExt)
		http.Error(w, "File extension doesn't match content type", http.StatusBadRequest)
		return
	}

	// Generate safe filename using UUID
	fileID := uuid.New().String()
	safeFilename := fileID + extension
	filePath := filepath.Join(UploadDirectory, safeFilename)

	// Create the destination file
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Set restrictive file permissions
	if err := dst.Chmod(0644); err != nil {
		log.Printf("Failed to set file permissions: %v", err)
	}

	// Copy file content with size limit
	_, err = io.CopyN(dst, file, MaxFileSize)
	if err != nil && err != io.EOF {
		log.Printf("Failed to save file: %v", err)
		os.Remove(filePath) // Clean up on failure
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	log.Printf("File uploaded successfully: %s (original: %s, type: %s)",
		safeFilename, header.Filename, contentType)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "filename": "%s", "file_id": "%s"}`,
		safeFilename, fileID)
}

func verifyMagicBytes(buffer []byte, contentType string) bool {
	magic, exists := magicBytes[contentType]
	if !exists {
		return false
	}

	if len(buffer) < len(magic) {
		return false
	}

	return bytes.HasPrefix(buffer, magic)
}

func isValidExtensionForType(ext, contentType string) bool {
	switch contentType {
	case "image/jpeg":
		return ext == ".jpg" || ext == ".jpeg"
	case "image/png":
		return ext == ".png"
	case "image/gif":
		return ext == ".gif"
	case "image/webp":
		return ext == ".webp"
	default:
		return false
	}
}
