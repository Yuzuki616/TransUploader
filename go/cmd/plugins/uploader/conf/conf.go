package conf

type SubLang struct {
	Lang    string   `json:"Lang"`
	Keyword []string `json:"Keyword"`
}
type MacCms struct {
	BaseUrl string `json:"BaseUrl"`
	Token   string `json:"Token"`
	TypeId  string `json:"TypeId"`
	Player  string `json:"Player"`
}
type Uploader struct {
	ApiType           string            `json:"ApiType"`
	Prefix            string            `json:"Prefix"`
	EnableTransCoding bool              `json:"EnableTransCoding"`
	Decoder           map[string]string `json:"Decoder"`
	Encoder           string            `json:"Encoder"`
	OutFile           string            `json:"OutFile"`
	ToHls             bool              `json:"ToHls"`
}

type Encode struct {
	Decoder map[string]string `json:"Decoder"`
	Encoder string            `json:"Encoder"`
}
