package maccms

import (
	"log"
	"testing"
)

func TestMacCms_UploadVideo(t *testing.T) {
	mc := New("IUEJXW8HTVPM78EJ", "http://moecg.net")
	err := mc.UploadVideo(364450, map[int]string{
		3: "http://moecg.net/1.mkv",
		4: "http://test.com/1.mkv",
	})
	log.Println(err)
}
