package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	var output bytes.Buffer

	ffmpeg_cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	ffmpeg_cmd.Stdout = &output
	err := ffmpeg_cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error processing video for fast start: %v", err)
	}

	return outputPath, nil
}
