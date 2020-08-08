package main

import (
	"bytes"
	"io/ioutil"

	"os"
	"testing"

	vod "eikcalb.dev/vod/src"
)

func TestImageResize(t *testing.T) {
	filename := "in.png"
	t.Log("Starting image resize test")
	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		t.Error(err.Error())
	}
	var out bytes.Buffer
	err = vod.ResizeImage(file, &out, vod.NewDimension(400, 480))
	if err != nil {
		t.Error(err.Error())
	}
	ioutil.WriteFile("out.png", out.Bytes(), os.ModePerm)
	t.Logf("Success!!")

}
