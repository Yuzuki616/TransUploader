package uploader

import (
	"CmsUploader/go/cmd/plugins/uploader/api/maccms"
	"CmsUploader/go/cmd/plugins/uploader/api/upload"
	"CmsUploader/go/cmd/plugins/uploader/api/upload/cos"
	"CmsUploader/go/cmd/plugins/uploader/common/ioprogress"
	"github.com/go-flutter-desktop/go-flutter/plugin"
	"sync"
)

const ChannelName = "yuzuki.io/ffmpeg"

type Uploader struct {
	maccmsApi *maccms.MacCms
	uploadApi upload.Upload
	Tasks     sync.Map // [string]*Task
	TaskLimit chan struct{}
}
type Task struct {
	Progress *ioprogress.Reader
	Status   string
}

func (u *Uploader) InitPlugin(messenger plugin.BinaryMessenger) error {
	u.uploadApi = cos.New("ap-shanghai")
	u.TaskLimit = make(chan struct{}, 10)
	channel := plugin.NewMethodChannel(messenger, ChannelName, plugin.JSONMethodCodec{})
	//channel.HandleFunc("transcoding", Transcoding)
	channel.HandleFunc("addUploadTask", u.AddUploadTask)
	channel.HandleFunc("listUploadTask", u.ListUploadTask)
	channel.HandleFunc("setCmsSettings", u.SetCmsSettings)
	channel.HandleFunc("setUploadInterfaceSettings", u.SetUploadInterfaceSettings)
	channel.HandleFunc("getUploadInterfaceSettings", u.GetUploadInterfaceSettings)
	return nil
}
