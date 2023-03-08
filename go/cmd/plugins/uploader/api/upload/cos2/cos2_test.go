package cos2

import (
	"log"
	"testing"
)

func TestCos2_UploadMultiPart(t *testing.T) {
	c := New("mkv")
	log.Println(c.UploadMultiPart("../../../1.mkv"))
}
