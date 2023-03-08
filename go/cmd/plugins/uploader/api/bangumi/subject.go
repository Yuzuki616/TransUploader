package bangumi

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"strconv"
	"strings"
	"time"
)

func GetDateRange(date string) (string, string) {
	t, _ := time.Parse("2006-01-02", date)
	return t.Add(-time.Hour * 24 * 10).Format("2006-01-02"),
		t.Add(time.Hour * 24 * 10).Format("2006-01-02")
}

type Filter struct {
	Type    []int    `json:"type"`
	Tag     []string `json:"tag"`
	AirDate []string `json:"air_date"`
	Rating  []string `json:"rating"`
	Rank    []string `json:"rank"`
	Nsfw    bool     `json:"nsfw"`
}
type SearchRequest struct {
	Keyword string `json:"keyword"`
	Sort    string `json:"sort"`
	Filter  Filter `json:"filter"`
}
type SubjectOverview struct {
	Date   string `json:"date"`
	Image  string `json:"image"`
	Name   string `json:"name"`
	NameCn string `json:"name_cn"`
	Tags   []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	} `json:"tags"`
	Score   float64 `json:"score"`
	Id      int64   `json:"id"`
	Rank    int     `json:"rank"`
	Summary string  `json:"summary"`
}
type SearchResponse struct {
	Data   []SubjectOverview `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

func (b *Bangumi) SearchSubject(keyword, date string) (*SubjectOverview, error) {
	start, end := GetDateRange(date)
	req := SearchRequest{
		Keyword: keyword,
		Sort:    "rank",
		Filter: Filter{
			Type: []int{2},
			AirDate: []string{
				">=" + start,
			},
		},
	}
	j, _ := json.Marshal(&req)
	rsp, err := resty.New().R().SetBody(j).Post("https://api.bgm.tv/v0/search/subjects")
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode() != 200 {
		return nil, errors.New("request failed")
	}
	searchRsp := SearchResponse{}
	_ = json.Unmarshal(rsp.Body(), &searchRsp)
	if len(searchRsp.Data) == 0 {
		return nil, NotFoundError
	}
	if searchRsp.Data[0].Date > end {
		for i := range searchRsp.Data {
			if searchRsp.Data[i].Date <= end {
				return &searchRsp.Data[i], nil
			}
		}
		return nil, NotFoundError
	}
	return &searchRsp.Data[0], nil
}

type GetSubjectInfoResponse struct {
	Id       int64  `json:"id"`
	Type     int    `json:"type"`
	Name     string `json:"name"`
	NameCn   string `json:"name_cn"`
	Summary  string `json:"summary"`
	Nsfw     bool   `json:"nsfw"`
	Locked   bool   `json:"locked"`
	Date     string `json:"date"`
	Platform string `json:"platform"`
	Images   struct {
		Large  string `json:"large"`
		Common string `json:"common"`
		Medium string `json:"medium"`
		Small  string `json:"small"`
		Grid   string `json:"grid"`
	} `json:"images"`
	Infobox []struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	} `json:"infobox"`
	Rating struct {
		Score float64 `json:"score"`
	} `json:"rating"`
	Volumes       int `json:"volumes"`
	Eps           int `json:"eps"`
	TotalEpisodes int `json:"total_episodes"`
	Collection    struct {
		Wish    int `json:"wish"`
		Collect int `json:"collect"`
		Doing   int `json:"doing"`
		OnHold  int `json:"on_hold"`
		Dropped int `json:"dropped"`
	} `json:"collection"`
	Tags []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	} `json:"tags"`
}

type SubjectInfo struct {
	Id       int64
	Name     string
	NameCn   string
	Summary  string
	Date     string
	Image    string
	Tags     []string
	Score    float64
	Actor    string
	Director string
	Writer   string
	Weekday  string
	Num      int
	TotalNum int
}

func trimBracket(s string) string {
	r := []rune(s)
	start := 0
	for i := 0; i < len(r); i++ {
		if r[i] == '(' || r[i] == '（' {
			start = i
		} else if r[i] == ')' || r[i] == '）' && start != 0 {
			r = append(r[:start], r[i+1:]...)
			i -= i - start
			start = 0
		}
	}
	return string(r)
}

func (b *Bangumi) GetSubjectInfo(id int64) (*SubjectInfo, error) {
	rsp, err := b.client.R().SetHeader("user-agent", "autoUpload/1.12").Get("https://api.bgm.tv/v0/subjects/" + strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode() != 200 {
		return nil, errors.New("request error: " + rsp.String())
	}
	rspObj := GetSubjectInfoResponse{}
	_ = json.Unmarshal(rsp.Body(), &rspObj)
	subjectInfo := SubjectInfo{
		Id:       rspObj.Id,
		Name:     rspObj.Name,
		NameCn:   rspObj.NameCn,
		Summary:  rspObj.Summary,
		Date:     rspObj.Date,
		Image:    rspObj.Images.Large,
		Tags:     make([]string, len(rspObj.Tags)),
		Score:    rspObj.Rating.Score,
		TotalNum: rspObj.TotalEpisodes,
	}
	for i := range rspObj.Tags {
		subjectInfo.Tags[i] = rspObj.Tags[i].Name
	}
	for i := range rspObj.Infobox {
		if rspObj.Infobox[i].Key == "导演" {
			subjectInfo.Director = trimBracket(rspObj.Infobox[i].Value.(string))
		} else if rspObj.Infobox[i].Key == "编剧" ||
			rspObj.Infobox[i].Key == "脚本" {
			subjectInfo.Writer = trimBracket(rspObj.Infobox[i].Value.(string))
		} else if rspObj.Infobox[i].Key == "主演" {
			subjectInfo.Actor = trimBracket(rspObj.Infobox[i].Value.(string))
		} else if rspObj.Infobox[i].Key == "放送星期" {
			subjectInfo.Weekday = strings.TrimPrefix(rspObj.Infobox[i].Value.(string), "星期")
		}
	}
	return &subjectInfo, nil
}

type PersonInfo struct {
	Images struct {
		Small  string `json:"small"`
		Grid   string `json:"grid"`
		Large  string `json:"large"`
		Medium string `json:"medium"`
	} `json:"images"`
	Name     string   `json:"name"`
	Relation string   `json:"relation"`
	Career   []string `json:"career"`
	Type     int      `json:"type"`
	Id       int      `json:"id"`
}

func (b *Bangumi) ListSubjectPerson(id int64) ([]PersonInfo, error) {
	rsp, err := b.client.R().Get("https://api.bgm.tv/v0/subjects/" + strconv.FormatInt(id, 10) + "/persons")
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode() != 200 {
		return nil, fmt.Errorf("request error: %s", rsp.String())
	}
	var person []PersonInfo
	_ = json.Unmarshal(rsp.Body(), &person)
	return person, nil
}

type CharacterInfo struct {
	Images struct {
		Small  string `json:"small"`
		Grid   string `json:"grid"`
		Large  string `json:"large"`
		Medium string `json:"medium"`
	} `json:"images"`
	Name     string `json:"name"`
	Relation string `json:"relation"`
	Actors   []struct {
		Images struct {
			Small  string `json:"small"`
			Grid   string `json:"grid"`
			Large  string `json:"large"`
			Medium string `json:"medium"`
		} `json:"images"`
		Name         string   `json:"name"`
		ShortSummary string   `json:"short_summary"`
		Career       []string `json:"career"`
		Id           int      `json:"id"`
		Type         int      `json:"type"`
		Locked       bool     `json:"locked"`
	} `json:"actors"`
	Type int `json:"type"`
	Id   int `json:"id"`
}
