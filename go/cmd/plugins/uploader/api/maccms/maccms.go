package maccms

import "CmsUploader/go/cmd/plugins/uploader/conf"

type MacCms struct {
	baseUrl string
	token   string
	typeId  string
	player  string
}

func New(c *conf.MacCms) *MacCms {
	return &MacCms{
		baseUrl: c.BaseUrl,
		token:   c.Token,
		typeId:  c.TypeId,
		player:  c.Player,
	}
}
