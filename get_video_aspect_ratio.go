package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type FFProbeOutput struct {
	Streams []Stream `json:"streams"`
}

type Stream struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running ffprobe: %w", err)
	}

	var output FFProbeOutput
	err := json.Unmarshal(stdout.Bytes(), &output)
	if err != nil {
		return "", err
	}

	if len(output.Streams) == 0 {
		return "", fmt.Errorf("no streams found in video")
	}

	// Get the first video stream
	stream := output.Streams[0]

	// Debug: Print actual dimensions
	fmt.Printf("Video dimensions: %dx%d\n", stream.Width, stream.Height)

	// Calculate decimal ratio for comparison
	decimalRatio := float64(stream.Width) / float64(stream.Height)
	fmt.Printf("Decimal ratio: %.4f\n", decimalRatio)

	r := gcd(stream.Width, stream.Height)
	aspectRatio := fmt.Sprintf("%d:%d", stream.Width/r, stream.Height/r)
	fmt.Printf("Calculated aspect ratio: %s\n", aspectRatio)

	// Use decimal ratio for more flexible matching
	ratio916 := 9.0 / 16.0 // 0.5625
	ratio169 := 16.0 / 9.0 // 1.7778

	tolerance := 0.01 // 1% tolerance - more precise

	fmt.Printf("9:16 target ratio: %.4f, difference: %.4f\n", ratio916, abs(decimalRatio-ratio916))
	fmt.Printf("16:9 target ratio: %.4f, difference: %.4f\n", ratio169, abs(decimalRatio-ratio169))

	if abs(decimalRatio-ratio916) < tolerance {
		return "9:16", nil
	} else if abs(decimalRatio-ratio169) < tolerance {
		return "16:9", nil
	}

	// Fallback to exact ratio matching
	switch aspectRatio {
	case "16:9", "9:16":
		return aspectRatio, nil
	default:
		return "other", nil
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func gcd(a, b int) int {
	if b == 0 {
		return a
	}
	return gcd(b, a%b)
}

func processVideoForFastStart(inputFilePath string) (string, error) {
	processedFilePath := fmt.Sprintf("%s.processing", inputFilePath)

	cmd := exec.Command("ffmpeg", "-i", inputFilePath, "-movflags", "faststart", "-codec", "copy", "-f", "mp4", processedFilePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error processing video: %s, %v", stderr.String(), err)
	}

	fileInfo, err := os.Stat(processedFilePath)
	if err != nil {
		return "", fmt.Errorf("could not stat processed file: %v", err)
	}
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("processed file is empty")
	}

	return processedFilePath, nil
}
