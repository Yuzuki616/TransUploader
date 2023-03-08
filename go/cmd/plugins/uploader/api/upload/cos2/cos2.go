package cos2

import (
	"CmsUploader/go/cmd/plugins/uploader/api/upload/cos"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

type Cos2 struct {
	client   *resty.Client
	fileType string
}

func New(fileType string) *Cos2 {
	return &Cos2{
		client:   resty.New().SetTimeout(5 * time.Minute).SetRetryCount(2),
		fileType: fileType,
	}
}

type GetCertRsp struct {
	CosUploadImgDomain string `json:"cos_upload_img_domain"`
	ServiceTime        int    `json:"service_time"`
	UploadFormat       string `json:"upload_format"`
	CosSessionToken    string `json:"cos_session_token"`
	CosRegionName      string `json:"cos_region_name"`
	CosSecretId        string `json:"cos_secret_id"`
	CosSecretKey       string `json:"cos_secret_key"`
	CosBucketName      string `json:"cos_bucket_name"`
	CosSwitch          bool   `json:"cosSwitch"`
	IsUploadOPic       int    `json:"isUploadOPic"`
}

func (c *Cos2) GetCert() (*GetCertRsp, error) {
	r, err := c.client.R().SetHeader("Content-Type", "application/json").
		SetBody(`{"data":{"img_domain":"filesystem1.hybbtree.com"}}`).
		Post("https://javaport.hybbtree.com/schoolweb/cos/getCosInfoForCrm")
	if err != nil {
		return nil, err
	} else if r.StatusCode() != 200 {
		return nil, errors.New(r.String())
	}
	cert := &GetCertRsp{}
	err = json.Unmarshal(r.Body(), cert)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal error: %v", err)
	}
	return cert, nil
}

func (c *Cos2) UploadMultiPart(file string) (string, error) {
	cert, err := c.GetCert()
	if err != nil {
		return "", err
	}
	c.client.SetPreRequestHook(func(c *resty.Client, req *http.Request) error {
		cos.AddAuthorizationHeader(cert.CosSecretId, cert.CosSecretKey, req, cos.NewAuthTime(time.Hour))
		req.Header.Set("x-cos-security-token", cert.CosSessionToken)
		return nil
	})
	defer func() {
		c.client.SetPreRequestHook(nil)
	}()
	baseUrl := "https://" + cert.CosBucketName + ".cos." + cert.CosRegionName + ".myqcloud.com/"
	webPath := "h5peple/images/" + time.Now().Format("20060102150405/") +
		hex.EncodeToString(md5.New().Sum([]byte(cert.CosSessionToken + cert.CosSecretId))[8:24]) + "." + c.fileType
	r, err := c.client.R().Post(baseUrl + webPath + "?uploads")
	if err != nil {
		return "", err
	} else if r.StatusCode() != 200 {
		return "", errors.New(r.String())
	}
	id := regexp.MustCompile(`<UploadId>(.*)</UploadId>`).FindStringSubmatch(r.String())
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("open file error: %v", err)
	}
	tmp := "<CompleteMultipartUpload>"
	tmpl := "\n<Part>\n<PartNumber>%d</PartNumber>\n<ETag>%s</ETag>\n</Part>"
	for n := 1; ; n++ {
		flag := false
		buf := make([]byte, 1024*1024*20)
		_, err = io.ReadFull(f, buf)
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
		r, err := c.client.R().SetBody(buf).Put(baseUrl + webPath + "?partNumber=" +
			strconv.Itoa(n) + "&uploadId=" + id[1])
		if err != nil {
			return "", fmt.Errorf("upload part %d error: %v", n, err)
		} else if r.StatusCode() != 200 {
			return "", fmt.Errorf("upload part %d error: %v", n, r.String())
		}
		tmp += fmt.Sprintf(tmpl, n, r.Header().Get("ETag"))
		log.Println(n, len(buf))
		if flag {
			tmp += "\n</CompleteMultipartUpload>"
			break
		}
	}
	r, err = c.client.R().SetBody(tmp).Post(baseUrl + webPath + "?uploadId=" + id[1])
	if err != nil {
		return "", fmt.Errorf("complete upload error: %v", err)
	} else if r.StatusCode() != 200 {
		return "", fmt.Errorf("complete upload error: %v", r.String())
	}
	return "https://bbtree-filesystem-1301933973.cos.ap-beijing.myqcloud.com/" + webPath, nil
}
