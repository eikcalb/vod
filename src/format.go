package vod

import (
	"errors"
)

// Dimension describes the length and height of image or video.
type Dimension struct {
	width  int
	height int
}

const (
	STOP = iota
)

var (
	// VideoSizes stores common output sizes for videos which will be used to differentiate quality.
	// The sizes assume the video aspect ratio is 16:9.
	// Input streams will be adjusted to match close to these values.
	//
	// Source for formats is: https://support.google.com/youtube/answer/6375112?co=GENIE.Platform%3DDesktop&hl=en
	VideoSizes map[string]Dimension = map[string]Dimension{
		"2160p": {3840, 2160},
		"1440p": {2560, 1440},
		"1080p": {1920, 1080},
		"720p":  {1280, 720},
		"480p":  {854, 480},
		"360p":  {640, 360},
		"240p":  {426, 240},
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