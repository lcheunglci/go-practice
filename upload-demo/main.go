package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	log.Println("Starting MediaGate server on :8080...")

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
    <form action="/upload" method="post" enctype="multipart/form-data">
        <label for="profile_image">Choose profile image:</label><br>
        <input type="file" id="profile_image" name="profile_image"><br><br>
        <input type="submit" value="Upload Image">
    </form>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form with no size limit (vulnerable!)
	err := r.ParseMultipartForm(0)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("profile_image")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Use original filename directly (vulnerable!)
	filePath := filepath.Join(uploadsDir, header.Filename)

	// Create the destination file
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy file content without any validation (vulnerable!)
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	log.Printf("File uploaded: %s", filePath)
	fmt.Fprintf(w, "File uploaded successfully: %s", header.Filename)
}
