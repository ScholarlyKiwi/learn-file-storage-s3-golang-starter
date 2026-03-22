package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	const expireTime = time.Hour * 24
	videoKey := strings.Split(*video.VideoURL, ",")
	if len(videoKey) < 2 {
		return video, fmt.Errorf("dbVideoToSignedVideo - insufficent keys")
	}
	presignedUrl, err := generatePresignedURL(cfg.s3client, videoKey[0], videoKey[1], expireTime)
	if err != nil {
		return video, err
	}
	video.VideoURL = &presignedUrl

	return video, nil
}
