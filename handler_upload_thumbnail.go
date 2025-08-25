package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	// Generate random 32-byte slice
	random32 := make([]byte, 32)
	_, err := rand.Read(random32)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate random bytes", err)
		return
	}

	// Encode to base64
	randomBase64 := base64.RawURLEncoding.EncodeToString(random32)
	_ = randomBase64 // Use the variable or remove if not needed

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20 // 10 MB

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse file", err)
		return
	}
	defer file.Close()

	image, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read image", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}

	// Check if user owns the video
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the owner of the video", nil)
		return
	}

	mediaType, types, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse media type", err)
		return
	}
	if len(types) > 0 {
		respondWithError(w, http.StatusBadRequest, "Unexpected media type parameters", nil)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	// Validate the file type (only allow image/jpeg andimage/png)

	var fileExtension string

	switch mediaType {
	case "image/jpeg":
		fileExtension = "jpg"
	case "image/png":
		fileExtension = "png"
	case "image/gif":
		fileExtension = "gif"
	case "image/webp":
		fileExtension = "webp"
	default:
		respondWithError(w, http.StatusBadRequest, "Unsupported file type", nil)
		return
	}

	filename := fmt.Sprintf("%s.%s", randomBase64, fileExtension)
	filepath := filepath.Join(cfg.assetsRoot, filename)

	destFile, err := os.Create(filepath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file", err)
		return
	}
	defer destFile.Close()
	// Write the image bytes to the file
	_, err = destFile.Write(image)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write file", err)
		return
	}
	io.Copy(destFile, file)
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	video.ThumbnailURL = &thumbnailURL

	// Update the video record in the database
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
