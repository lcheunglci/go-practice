package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxFileSize         = 5 * 1024 * 1024 // 5MB
	UploadDirectory     = "/var/lib/mediagate/uploads"
	TempDirectory       = "/var/lib/mediagate/temp"
	QuarantineDirectory = "/var/lib/mediagate/quarantine"
	ClamAVSocketPath    = "/var/run/clamav/clamd.ctl"
	ScanTimeout         = 30 * time.Second
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

	// Ensure directories exist with proper permissions
	if err := setupDirectories(); err != nil {
		log.Fatal("Failed to setup directories:", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("POST /upload", uploadHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func setupDirectories() error {
	dirs := []string{UploadDirectory, TempDirectory, QuarantineDirectory}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		log.Printf("Directory created/verified: %s", dir)
	}

	return nil
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
    <p><em>All uploads are scanned for malware</em></p>
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
	// Create context with timeout for the entire upload process
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

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

	// Save to temporary directory first for scanning
	tempFilePath := filepath.Join(TempDirectory, safeFilename)

	// Create the temporary file with restrictive permissions
	tempFile, err := os.OpenFile(tempFilePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		log.Printf("Failed to create temp file: %v", err)
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFilePath) // Clean up temp file

	// Copy file content with size limit
	_, err = io.CopyN(tempFile, file, MaxFileSize)
	if err != nil && err != io.EOF {
		log.Printf("Failed to save temp file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Close temp file before scanning
	tempFile.Close()

	// Scan file with ClamAV
	scanResult, err := scanFileWithClamAV(ctx, tempFilePath)
	if err != nil {
		log.Printf("Virus scan failed: %v", err)
		http.Error(w, "Virus scan failed", http.StatusInternalServerError)
		return
	}

	if scanResult.Infected {
		// Move infected file to quarantine
		quarantinePath := filepath.Join(QuarantineDirectory, safeFilename)
		if err := moveFile(tempFilePath, quarantinePath); err != nil {
			log.Printf("Failed to quarantine infected file: %v", err)
		}

		log.Printf("SECURITY ALERT: Infected file quarantined: %s (original: %s, threat: %s)",
			safeFilename, header.Filename, scanResult.ThreatName)

		http.Error(w, "File contains malware and has been quarantined", http.StatusBadRequest)
		return
	}

	// File is clean, move to final destination
	finalFilePath := filepath.Join(UploadDirectory, safeFilename)
	if err := moveFile(tempFilePath, finalFilePath); err != nil {
		log.Printf("Failed to move clean file to uploads: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	log.Printf("File uploaded and scanned successfully: %s (original: %s, type: %s)",
		safeFilename, header.Filename, contentType)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "filename": "%s", "file_id": "%s", "scanned": true}`,
		safeFilename, fileID)
}

type ScanResult struct {
	Infected   bool
	ThreatName string
}

func scanFileWithClamAV(ctx context.Context, filePath string) (*ScanResult, error) {
	// Create context with scan timeout
	scanCtx, cancel := context.WithTimeout(ctx, ScanTimeout)
	defer cancel()

	// Use clamdscan for better performance with clamd daemon
	cmd := exec.CommandContext(scanCtx, "clamdscan", "--fdpass", "--no-summary", filePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Parse clamdscan exit codes
	result := &ScanResult{}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			switch exitError.ExitCode() {
			case 1:
				// Infected file found
				result.Infected = true
				// Extract threat name from output
				output := stdout.String()
				if idx := strings.Index(output, "FOUND"); idx != -1 {
					parts := strings.Fields(output[:idx])
					if len(parts) > 0 {
						result.ThreatName = parts[len(parts)-1]
					}
				}
				return result, nil
			case 2:
				// Error occurred
				return nil, fmt.Errorf("clamdscan error: %s", stderr.String())
			default:
				return nil, fmt.Errorf("clamdscan failed with exit code %d: %s",
					exitError.ExitCode(), stderr.String())
			}
		}
		return nil, fmt.Errorf("failed to run clamdscan: %w", err)
	}

	// Exit code 0 means clean file
	result.Infected = false
	return result, nil
}

func moveFile(src, dst string) error {
	// Try atomic move first (works if on same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// If rename fails, copy and delete
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		os.Remove(dst) // Clean up on failure
		return err
	}

	// Remove source file after successful copy
	return os.Remove(src)
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
