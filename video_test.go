package main

import (
	"os"
	"testing"

	vod "eikcalb.dev/vod/src"
)

func TestVideoResize(t *testing.T) {
	filename := "file_example_MP4_1920_18MG.mp4"
	t.Log("Starting video resize test")
	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		t.Error(err.Error())
	}
	err = vod.ProcessVideoInput(file)
	if err != nil {
		t.Error(err.Error())
	} else {
		t.Logf("Success!!")
	}
}
