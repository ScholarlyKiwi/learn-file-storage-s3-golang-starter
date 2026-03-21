package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type jsonffprobeOutput struct {
	Streams []jsonStreams
}
type jsonStreams struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func getVideoAspectRatio(filePath string) (string, error) {

	var output bytes.Buffer

	ffprobe_cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	ffprobe_cmd.Stdout = &output
	err := ffprobe_cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running ffprobe: %v", err)
	}

	var jsonOutput jsonffprobeOutput
	err = json.Unmarshal(output.Bytes(), &jsonOutput)

	width := jsonOutput.Streams[0].Width
	height := jsonOutput.Streams[0].Height

	if width/16 == height/9 {
		return "16:9", nil
	} else if width/9 == height/16 {
		return "9:16", nil
	}

	return "other", nil
}
