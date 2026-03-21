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
	"slices"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusRequestHeaderFieldsTooLarge, "Error parsing form", err)
		return
	}

	thumbnail_file, thumbnail_header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid reading thumbnail", err)
		return
	}
	defer thumbnail_file.Close()
	mediaType := thumbnail_header.Header.Get("Content-Type")
	parsedMediaType, _, err := mime.ParseMediaType(mediaType)

	allowedType := []string{
		"image/jpeg",
		"image/png",
	}

	if !slices.Contains(allowedType, parsedMediaType) {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Invalid media type: %v", parsedMediaType), fmt.Errorf("Invalid media type: %v", parsedMediaType))
		return
	}

	video_metadata, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error retrieving video metavideo: %v", err)
		return
	}

	if video_metadata.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the author.", fmt.Errorf("UserID does not match authorID"))
		return
	}

	extensions, err := mime.ExtensionsByType(mediaType)
	if err != nil || len(extensions) == 0 {
		respondWithError(w, http.StatusInternalServerError, "Error determining extension", err)
		return
	}

	random_filename := make([]byte, 32)
	rand.Read(random_filename)
	filename := base64.URLEncoding.EncodeToString(random_filename)
	thumbnail_filename := fmt.Sprintf("%v%v", filename, extensions[0])
	thumbnail_filepath := filepath.Join(cfg.assetsRoot, thumbnail_filename)

	file_on_disc, err := os.Create(thumbnail_filepath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error create temp file", err)
		return
	}
	defer file_on_disc.Close()

	size, err := io.Copy(file_on_disc, thumbnail_file)
	if err != nil || size == 0 {
		respondWithError(w, http.StatusInternalServerError, "Error writing to disk", err)
		return
	}

	thumbnail_url := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, thumbnail_filename)
	video_metadata.ThumbnailURL = &thumbnail_url

	err = cfg.db.UpdateVideo(video_metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Updating Video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video_metadata)
}
