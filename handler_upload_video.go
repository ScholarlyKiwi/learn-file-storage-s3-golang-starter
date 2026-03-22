package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"slices"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

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

	fmt.Println("uploading video", videoID, "by user", userID)

	const maxMemory = 10 << 30

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusRequestHeaderFieldsTooLarge, "Error parsing form", err)
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

	video_file, video_header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error uploading video file", err)
		return
	}
	defer video_file.Close()

	mediaType := video_header.Header.Get("Content-Type")
	parsedMediaType, _, err := mime.ParseMediaType(mediaType)

	allowedType := []string{
		"video/mp4",
	}

	if !slices.Contains(allowedType, parsedMediaType) {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Invalid media type: %v", parsedMediaType), fmt.Errorf("Invalid media type: %v", parsedMediaType))
		return
	}

	random_filename := make([]byte, 32)
	rand.Read(random_filename)
	tmp_filename := base64.URLEncoding.EncodeToString(random_filename)

	tempfile, err := os.CreateTemp("", tmp_filename)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create tempfile", err)
		return
	}
	defer os.Remove(tempfile.Name())
	defer tempfile.Close()

	_, err = io.Copy(tempfile, video_file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error copying to tempfile", err)
		return
	}

	_, err = tempfile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error resetting video reading", err)
		return
	}

	processFilePath, err := processVideoForFastStart(tempfile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing file for upload", err)
		return
	}
	processFile, err := os.Open(processFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error opening processed file", err)
		return
	}
	defer os.Remove(processFile.Name())

	aspectRatio, err := getVideoAspectRatio(processFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error calculating aspect ratio", err)
		return
	}

	var prefix string
	switch aspectRatio {
	case "16:9":
		prefix = "landscape/"
	case "9:16":
		prefix = "portrait/"
	default:
		prefix = "other/"
	}

	s3Key := prefix + tmp_filename

	putObjParam := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &s3Key,
		Body:        processFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3client.PutObject(r.Context(), &putObjParam)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error sending video to S3", err)
		return
	}

	videoURL := fmt.Sprintf("%v.s3.%v.amazonaws.com,%v", cfg.s3Bucket, cfg.s3Region, s3Key)
	video_metadata.VideoURL = &videoURL
	video_metadata, err = cfg.dbVideoToSignedVideo(video_metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error signing video.", err)
		return
	}

	err = cfg.db.UpdateVideo(video_metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Updating Video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video_metadata)
}
