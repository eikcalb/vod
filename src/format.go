package vod

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// Dimension describes the length and height of image or video.
type Dimension struct {
	width  int
	height int
}

const (
	// STOP stops iteration
	STOP = iota
)

var (
	// VideoSizes stores common output sizes for videos which will be used to differentiate quality.
	// The sizes assume the video aspect ratio is 9:16.
	// Input streams will be adjusted to match close to these values.
	//
	// Source for formats is: https://support.google.com/youtube/answer/6375112?co=GENIE.Platform%3DDesktop&hl=en
	VideoSizes map[string]Dimension = map[string]Dimension{
		"2160p": {2160, 3840},
		"1440p": {1440, 2560},
		"1080p": {1080, 1920},
		"720p":  {720, 1280},
		"480p":  {480, 854},
		"360p":  {360, 640},
		"240p":  {240, 426},
	}

	// VideoArray is an ordered list of display sizes supported
	VideoArray = []int{
		2160,
		1440,
		1080,
		720,
		480,
		360,
		240,
	}
)

// NewDimension returns a pointer to a Dimension instance
func NewDimension(w, h int) *Dimension {
	return &Dimension{w, h}
}

// FindNearestNext returns the index for which to get the next video resolution
func (d Dimension) FindNearestNext(cur int) (int, error) {
	if cur == 0 {
		// Check if the dimension is larger than maximum
		if d.height >= VideoArray[0] {
			return 2, nil
		} else if d.height >= VideoArray[1] {
			return 3, nil
		} else if d.height >= VideoArray[2] {
			return 3, nil
		} else if d.height >= VideoArray[3] {
			return 4, nil
		} else {
			return 6, nil
		}
	}
	nextVal := cur + 2
	if nextVal >= 5 {
		return STOP, errors.New("No more size available")
	}
	return nextVal, nil
}

// IsVideo checks if the provided file header is a video
func IsVideo(data []byte) (bool, string) {
	detectedType := http.DetectContentType(data)
	if strings.EqualFold(detectedType, "application/octet-stream") {
		// If the type is undetecteable, return false
		return false, "video/mp4"
	} else if strings.HasPrefix(detectedType, "video") {
		return true, detectedType
	} else {
		return false, detectedType
	}
}

func getMediaFilePath(input string) string {
	pathArray := strings.Split(input, "/")
	// uuid will be the second entry in the array
	return "media/" + pathArray[1]
}

func getCatalogueFilePath(input string) string {
	pathArray := strings.Split(input, "/")
	// uuid will be the second entry in the array
	return "catalogue/" + pathArray[1]
}

func generatePath(prefix string) string {
	return prefix + uuid.New().String()
}
