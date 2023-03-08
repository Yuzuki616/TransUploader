package cos

import (
	"CmsUploader/go/cmd/plugins/uploader/common/ioprogress"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Cos struct {
	client *resty.Client
	region string
}

func New(region string) *Cos {
	return &Cos{
		client: resty.New().
			SetTimeout(5 * time.Minute).
			SetRetryCount(2).
			SetPreRequestHook(func(c *resty.Client, r *http.Request) error {
				AddAuthorizationHeader(r, NewAuthTime(time.Hour))
				return nil
			}),
		region: region,
	}
}

func (c *Cos) Upload(file string, reader *ioprogress.Reader) (string, error) {
	f, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	f2, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f2.Close()
	reader.ChangeReader(f2)
	url := ""
	if f.Size() <= 104857600 {
		url, err = c.UploadSimple(reader, strings.TrimPrefix(path.Ext(file), "."))
	} else {
		url, err = c.UploadMultipart(reader, strings.TrimPrefix(path.Ext(file), "."))
	}
	return strings.ReplaceAll(url, "http", "https"), err
}

func (c *Cos) GetSignature() (string, error) {
	r, err := c.client.R().Get("https://editor.futunn.com/video-signature?lang=zh-cn")
	if err != nil {
		return "", err
	}
	jsonR := struct {
		Signature string `json:"signature"`
	}{}
	err = json.Unmarshal(r.Body(), &jsonR)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error: %v", err)
	}
	return jsonR.Signature, nil
}

type GetCertRsp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Video struct {
			StorageSignature string `json:"storageSignature"`
			StoragePath      string `json:"storagePath"`
		} `json:"video"`
		StorageAppId    int    `json:"storageAppId"`
		StorageBucket   string `json:"storageBucket"`
		StorageRegion   string `json:"storageRegion"`
		StorageRegionV5 string `json:"storageRegionV5"`
		Domain          string `json:"domain"`
		VodSessionKey   string `json:"vodSessionKey"`
		TempCertificate struct {
			SecretId    string `json:"secretId"`
			SecretKey   string `json:"secretKey"`
			Token       string `json:"token"`
			ExpiredTime int    `json:"expiredTime"`
		} `json:"tempCertificate"`
		AppId                     int    `json:"appId"`
		Timestamp                 int    `json:"timestamp"`
		StorageRegionV51          string `json:"StorageRegionV5"`
		MiniProgramAccelerateHost string `json:"MiniProgramAccelerateHost"`
	} `json:"data"`
}
type CommitUploadRsp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Video struct {
			Url           string `json:"url"`
			VerifyContent string `json:"verify_content"`
		} `json:"video"`
		FileId string `json:"fileId"`
	} `json:"data"`
}

func (c *Cos) GetCert(fileType string) (*GetCertRsp, error) {
	sign, err := c.GetSignature()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"signature":     sign,
		"videoName":     "finance",
		"storageRegion": c.region,
		"videoSize":     1,
		"videoType":     fileType,
	})
	r, err := c.client.R().SetBody(reqBody).Post("https://vod2.qcloud.com/v3/index.php?Action=ApplyUploadUGC")
	if err != nil {
		return nil, fmt.Errorf("get cert error: %v", err)
	} else if r.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get cert error: %v", r.String())
	}
	cert := &GetCertRsp{}
	err = json.Unmarshal(r.Body(), cert)
	if cert.Code != 0 {
		return nil, fmt.Errorf("get cert error: %v", cert.Message)
	}
	if err != nil {
		return nil, fmt.Errorf("json unmarshal error: %v", err)
	}
	return cert, nil
}
func (c *Cos) UploadSimple(reader io.Reader, fileType string) (string, error) {
	cert, err := c.GetCert(fileType)
	if err != nil {
		return "", err
	}
	r, err := c.client.R().SetHeaders(map[string]string{
		"SecretId":             cert.Data.TempCertificate.SecretId,
		"SecretKey":            cert.Data.TempCertificate.SecretKey,
		"x-cos-security-token": cert.Data.TempCertificate.Token,
	}).SetBody(reader).Put("https://" + cert.Data.StorageBucket + "-" + strconv.Itoa(cert.Data.StorageAppId) +
		".cos." + cert.Data.StorageRegionV5 + ".myqcloud.com" + cert.Data.Video.StoragePath)
	if err != nil {
		return "", err
	} else if r.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("upload error: %v", r.String())
	}
	return c.CommitUpload(cert.Data.VodSessionKey, cert)
}

func (c *Cos) UploadMultipart(reader io.Reader, fileType string) (string, error) {
	cert, err := c.GetCert(fileType)
	if err != nil {
		return "", err
	}
	baseUrl := "https://" + cert.Data.StorageBucket + "-" +
		strconv.Itoa(cert.Data.StorageAppId) +
		".cos." + cert.Data.StorageRegionV5 + ".myqcloud.com"
	r, err := c.client.R().SetHeaders(map[string]string{
		"SecretId":             cert.Data.TempCertificate.SecretId,
		"SecretKey":            cert.Data.TempCertificate.SecretKey,
		"x-cos-security-token": cert.Data.TempCertificate.Token,
	}).Post(baseUrl + cert.Data.Video.StoragePath + "?uploads")
	if err != nil {
		return "", fmt.Errorf("get upload id error: %v", err)
	} else if r.StatusCode() != 200 {
		return "", fmt.Errorf("get upload id error: %v", r.String())
	}
	id := regexp.MustCompile(`<UploadId>(.*)</UploadId>`).FindStringSubmatch(r.String())
	tmp := "<CompleteMultipartUpload>"
	tmpl := "\n<Part>\n<PartNumber>%d</PartNumber>\n<ETag>%s</ETag>\n</Part>"
	for n := 1; ; n++ {
		flag := false
		buf := make([]byte, 1024*1024*20)
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				return "", fmt.Errorf("read file error: %v", err)
			} else {
				if len(buf) == 0 {
					break
				} else {
					flag = true
				}
			}
		}
		r, err := c.client.R().SetHeaders(map[string]string{
			"SecretId":             cert.Data.TempCertificate.SecretId,
			"SecretKey":            cert.Data.TempCertificate.SecretKey,
			"x-cos-security-token": cert.Data.TempCertificate.Token,
		}).SetBody(buf).Put(baseUrl + cert.Data.Video.StoragePath + "?partNumber=" +
			strconv.Itoa(n) + "&uploadId=" + id[1])
		if err != nil {
			return "", fmt.Errorf("upload part %d error: %s", n, err)
		} else if r.StatusCode() != 200 {
			return "", fmt.Errorf("upload part %d error: %s", n, r)
		}
		tmp += fmt.Sprintf(tmpl, n, r.Header().Get("ETag"))
		if flag {
			tmp += "\n</CompleteMultipartUpload>"
			break
		}
	}
	r, err = c.client.R().SetHeaders(map[string]string{
		"SecretId":             cert.Data.TempCertificate.SecretId,
		"SecretKey":            cert.Data.TempCertificate.SecretKey,
		"x-cos-security-token": cert.Data.TempCertificate.Token,
	}).SetBody(tmp).Post(baseUrl + cert.Data.Video.StoragePath + "?uploadId=" + id[1])
	if err != nil {
		return "", fmt.Errorf("complete upload error: %v", err)
	} else if r.StatusCode() != 200 {
		return "", fmt.Errorf("complete upload error: %v", r.String())
	}
	return c.CommitUpload(cert.Data.VodSessionKey, cert)
}

func (c *Cos) CommitUpload(session string, cert *GetCertRsp) (string, error) {
	sign, err := c.GetSignature()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"signature":     sign,
		"vodSessionKey": session,
	})
	r, err := c.client.R().SetHeaders(map[string]string{
		"SecretId":             cert.Data.TempCertificate.SecretId,
		"SecretKey":            cert.Data.TempCertificate.SecretKey,
		"x-cos-security-token": cert.Data.TempCertificate.Token,
	}).SetBody(reqBody).Post("https://vod2.qcloud.com/v3/index.php?Action=CommitUploadUGC")
	if err != nil {
		return "", fmt.Errorf("commit upload error: %v", err)
	} else if r.StatusCode() != 200 {
		return "", fmt.Errorf("commit upload error: %v", r.String())
	}
	commit := CommitUploadRsp{}
	err = json.Unmarshal(r.Body(), &commit)
	if err != nil {
		return "", fmt.Errorf("unmarshal commit upload response error: %v", err)
	} else if commit.Code != 0 {
		return "", fmt.Errorf("commit upload error: %v", commit.Message)
	}
	return commit.Data.Video.Url, nil
}
