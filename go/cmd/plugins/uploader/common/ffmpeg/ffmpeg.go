package ffmpeg

import (
	"github.com/u2takey/ffmpeg-go"
	"os"
	"path"
)

func SegmentationVideoToHls(videoPath, outPath string) error {
	if _, err := os.Stat(path.Join(outPath, "ts")); os.IsNotExist(err) {
		os.MkdirAll(path.Join(outPath, "ts"), os.ModePerm)
	}
	err := ffmpeg_go.Input(videoPath).Output(path.Join(outPath, "ts", "index.m3u8"), ffmpeg_go.KwArgs{
		"c":             "copy",
		"sn":            "",
		"f":             "hls",
		"hls_time":      "3",
		"hls_list_size": "0",
	}).OverWriteOutput().ErrorToStdOut().Run()
	if err != nil {
		return err
	}
	os.Rename(path.Join(outPath, "ts", "index.m3u8"), path.Join(outPath, "index.m3u8"))
	return nil
}
