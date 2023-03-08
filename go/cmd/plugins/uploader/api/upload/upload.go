package upload

import (
	"CmsUploader/go/cmd/plugins/uploader/common/ioprogress"
)

type Upload interface {
	Upload(file string, reader *ioprogress.Reader) (string, error)
}
