package main

import (
	"fmt"
	"io"
	"net/http"

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
	media_type := thumbnail_header.Header.Get("Content-Type")
	thumbnail_bytes, err := io.ReadAll(thumbnail_file)

	video_metadata, err := cfg.db.GetVideo(videoID)

	if video_metadata.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the author.", fmt.Errorf("UserID does not match authorID"))
	}

	thumbnail := thumbnail{
		data:      thumbnail_bytes,
		mediaType: media_type,
	}
	videoThumbnails[videoID] = thumbnail

	thumbnail_url := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", cfg.port, videoIDString)
	video_metadata.ThumbnailURL = &thumbnail_url

	err = cfg.db.UpdateVideo(video_metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Updating Video", err)
	}

	respondWithJSON(w, http.StatusOK, video_metadata)
}
