package maccms

import (
	"CmsUploader/go/cmd/plugins/uploader/api/bangumi"
	"fmt"
	"github.com/go-resty/resty/v2"
	"log"
	"strconv"
	"strings"
)

type UploadVideoRequest struct {
	Pass             string
	TypeId           string
	VideoName        string
	VideoPic         string
	VideoEn          string
	VideoStatus      string
	VodArea          string
	VodLang          string
	VideoDoubanId    string
	VideoContent     string
	VideoDoubanScore string
	VideoPubdate     string
	VideoActor       string
	VideoDirector    string
	VideoWriter      string
	VideoYear        string
	VideoTags        []string
	VideoWeekday     string
	VideoPlayFrom    string
	VideoPlayUrl     map[int]string
	VideoTotal       string
	VideoSerial      string
}

func (v *UploadVideoRequest) Map() map[string]string {
	tmp := make(map[string]string, 23)
	tmp["pass"] = v.Pass
	tmp["type_id"] = v.TypeId
	tmp["vod_name"] = v.VideoName
	tmp["vod_pic"] = v.VideoPic
	tmp["vod_en"] = v.VideoEn
	tmp["vod_status"] = v.VideoStatus
	tmp["vod_area"] = v.VodArea
	tmp["vod_lang"] = v.VodLang
	tmp["vod_tag"] = strings.Join(v.VideoTags, " ")
	tmp["vod_class"] = "动画"
	tmp["vod_weekday"] = v.VideoWeekday
	tmp["vod_douban_id"] = v.VideoDoubanId
	tmp["vod_content"] = v.VideoContent
	tmp["vod_actor"] = v.VideoActor
	tmp["vod_director"] = v.VideoDirector
	tmp["vod_writer"] = v.VideoWriter
	tmp["vod_play_from"] = v.VideoPlayFrom
	url := ""
	for i := range v.VideoPlayUrl {
		if url != "" {
			url += "#"
		}
		url += "第" + strconv.Itoa(i) + "集$" + v.VideoPlayUrl[i]
	}
	tmp["vod_play_url"] = url
	tmp["vod_douban_score"] = v.VideoDoubanScore
	tmp["vod_pubdate"] = v.VideoPubdate
	tmp["vod_year"] = v.VideoYear
	tmp["vod_total"] = v.VideoTotal
	tmp["vod_serial"] = v.VideoSerial
	return tmp
}

func getMaxKey(url map[int]string) int {
	var max int
	for k := range url {
		if k > max {
			max = k
		}
	}
	return max
}

func (m *MacCms) UploadVideo(bgmId int64, playUrl map[int]string) error {
	bgm := bangumi.New()
	sub, err := bgm.GetSubjectInfo(bgmId)
	if err != nil {
		return fmt.Errorf("get subject error: %w", err)
	}
	year := ""
	if sub.Date != "" {
		year = sub.Date[:4]
	}
	form := &UploadVideoRequest{
		Pass:             m.token,
		TypeId:           m.typeId,
		VideoName:        sub.NameCn,
		VideoPic:         sub.Image,
		VideoEn:          sub.Name,
		VideoStatus:      "0",
		VodArea:          "日本",
		VodLang:          "日语",
		VideoTags:        sub.Tags,
		VideoWeekday:     sub.Weekday,
		VideoDoubanId:    strconv.FormatInt(bgmId, 10),
		VideoContent:     sub.Summary,
		VideoActor:       sub.Actor,
		VideoDirector:    sub.Director,
		VideoWriter:      sub.Writer,
		VideoPlayFrom:    m.player,
		VideoPlayUrl:     playUrl,
		VideoDoubanScore: strconv.FormatFloat(sub.Score, 'f', 1, 64),
		VideoPubdate:     sub.Date,
		VideoYear:        year,
		VideoTotal:       strconv.Itoa(sub.TotalNum),
		VideoSerial:      strconv.Itoa(getMaxKey(playUrl)),
	}
	rsp, err := resty.New().R().SetFormData(form.Map()).Post(m.baseUrl + "/api.php/receive/vod")
	//log.Println(rsp)
	if err != nil {
		return fmt.Errorf("upload video error: %s", err)
	} else if rsp.StatusCode() != 200 {
		return fmt.Errorf("upload video error: %s", rsp.String())
	}
	log.Println(rsp)
	return nil
}
