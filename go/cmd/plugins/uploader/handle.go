package uploader

import (
	"CmsUploader/go/cmd/plugins/uploader/api/maccms"
	"CmsUploader/go/cmd/plugins/uploader/api/upload/cos"
	"CmsUploader/go/cmd/plugins/uploader/common/ffmpeg"
	"CmsUploader/go/cmd/plugins/uploader/common/file"
	"CmsUploader/go/cmd/plugins/uploader/common/ioprogress"
	"CmsUploader/go/cmd/plugins/uploader/conf"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	path2 "path"
	"strconv"
	"strings"
)

func (u *Uploader) GetUploadInterfaceSettings(interface{}) (reply interface{}, err error) {
	return map[string][]string{
		"cos": {
			"region",
			"fileType",
		},
		"cos2": {
			"region",
			"fileType",
		},
		"jd": {},
		"ld": {},
	}, nil
}

func getArgs(args interface{}) (map[string]string, error) {
	arg := make(map[string]string)
	err := json.Unmarshal(args.(json.RawMessage), &arg)
	if err != nil {
		return nil, err
	}
	/*if a, ok := args.(map[interface{}]interface{}); ok {
		for k, v := range a {
			if ks, ok := k.(string); ok {
				if vs, ok := v.(string); ok {
					arg[ks] = vs
				} else if v != nil {
					return nil, fmt.Errorf("%s arg is not vail", k)
				}
			} else if v != nil {
				return nil, fmt.Errorf("%s arg is not vail", k)
			}
		}
	} else {
		return nil, errors.New("args type is not vail")
	}*/
	return arg, nil
}

func (u *Uploader) SetUploadInterfaceSettings(args interface{}) (reply interface{}, err error) {
	arg, err := getArgs(args)
	if err != nil {
		return
	}
	switch arg["type"] {
	case "cos":
		u.uploadApi = cos.New(arg["region"])
	}
	return nil, nil
}

func (u *Uploader) SetCmsSettings(args interface{}) (reply interface{}, err error) {
	arg, err := getArgs(args)
	if arg["baseUrl"] == "" ||
		arg["token"] == "" ||
		arg["typeId"] == "" ||
		arg["player"] == "" {
		return nil, errors.New("args is not vail")
	}
	u.maccmsApi = maccms.New(&conf.MacCms{
		BaseUrl: arg["baseUrl"],
		Token:   arg["token"],
		TypeId:  arg["typeId"],
		Player:  arg["player"],
	})
	if err != nil {
		return
	}
	return nil, nil
}

func (u *Uploader) uploadTask(path string, id int64, ep int, isSlice bool) {
	u.TaskLimit <- struct{}{}
	var err error
	defer func() {
		<-u.TaskLimit
		t, _ := u.Tasks.Load(path2.Base(path))
		t.(*Task).Status = "上传完成"
		if err != nil {
			t.(*Task).Status = "上传失败"
		}
	}()
	if isSlice {
		progress := ioprogress.NewReader(nil, 1)
		u.Tasks.Store(path2.Base(path), &Task{
			Status:   "切片中",
			Progress: progress,
		})
		err = ffmpeg.SegmentationVideoToHls(path, path2.Join("./m3u8Out", path2.Base(path)))
		if err != nil {
			log.Println("segmentation video error: ", err)
			return
		}
		size, err2 := file.GetFileOrDirSize(path)
		if err != nil {
			log.Println("get folder size error: ", err2)
			err = err2
			return
		}
		t, _ := u.Tasks.Load(path2.Base(path))
		t.(*Task).Status = "上传中"
		t.(*Task).Progress.Total = size
		f, err2 := os.OpenFile(path2.Join("./m3u8Out", path2.Base(path), "index.m3u8"), os.O_RDWR, 0666)
		if err2 != nil {
			log.Println("open index.m3u8 error: ", err2)
			err = err2
			return
		}
		defer f.Close()
		indexB, err2 := io.ReadAll(f)
		if err2 != nil {
			log.Println("read index.m3u8 error: ", err2)
			return
		}
		index := strings.Split(string(indexB), "\n")
		for i := 0; i < len(index); i++ {
			if strings.HasSuffix(index[i], ".ts") {
				url, err3 := u.uploadApi.Upload(path2.Join("./m3u8Out", path2.Base(path), "ts", index[i]), progress)
				if err3 != nil {
					log.Println("upload file error: ", err3)
					err = err3
					return
				}
				index[i] = url
			}
		}
		f.Seek(0, 0)
		_, err2 = f.Write([]byte(strings.Join(index, "\n")))
		if err2 != nil {
			log.Println("write index.m3u8 error: ", err2)
			err = err2
			return
		}
		os.RemoveAll(path2.Join("./m3u8Out", path2.Base(path), "ts"))
	} else {
		var size int64
		size, err = file.GetFileOrDirSize(path)
		if err != nil {
			log.Println("get file size error: ", err)
			return
		}
		progress := ioprogress.NewReader(nil, size)
		u.Tasks.Store(path2.Base(path), &Task{
			Status:   "上传中",
			Progress: progress,
		})
		var s string
		s, err = u.uploadApi.Upload(path, progress)
		if err != nil {
			log.Printf("upload %s err: %s", path, err)
			return
		}
		err = u.maccmsApi.UploadVideo(id, map[int]string{
			ep: s,
		})
		if err != nil {
			log.Printf("upload %s to cms err: %s", path, err)
		}
	}
}

func (u *Uploader) AddUploadTask(args interface{}) (reply interface{}, err error) {
	arg, err := getArgs(args)
	if err != nil {
		return nil, err
	}
	if u.maccmsApi == nil {
		return nil, errors.New("maccms is not init")
	}
	if u.uploadApi == nil {
		return nil, errors.New("upload is not init")
	}
	id, _ := strconv.ParseInt(arg["bangumiId"], 10, 64)
	ep, _ := strconv.Atoi(arg["episode"])
	if arg["isSlice"] == "1" {
		go u.uploadTask(arg["path"], id, ep, true)
	} else {
		go u.uploadTask(arg["path"], id, ep, false)
	}
	return nil, nil
}

func (u *Uploader) ListUploadTask(interface{}) (reply interface{}, err error) {
	tasks := make([]map[string]string, 0)
	u.Tasks.Range(func(key, value interface{}) bool {
		tasks = append(tasks, map[string]string{
			"name": key.(string),
			"status": value.(*Task).Status +
				" " + value.(*Task).Progress.Progress(),
		})
		if value.(*Task).Status == "上传完成" {
			u.Tasks.Delete(key)
		}
		return true
	})
	return tasks, nil
}
