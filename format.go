package vod

type Dimension struct {
	width  int
	height int
}

var (
	// VideoSizes stores common output sizes for videos which will be used to differentiate quality.
	// The sizes assume the video aspect ratio is 4:3.
	// Input streams will be adjusted to match close to these values.
	VideoSizes map[string]Dimension = map[string]Dimension{
		"2160p": {3840, 2160},
		"1440p": {2560, 1440},
		"1080p": {1920, 1080},
		"720p":  {1280, 720},
		"480p":  {854, 480},
		"360p":  {640, 360},
		"240p":  {426, 240},
		"140p":  {248, 140},
	}
)
