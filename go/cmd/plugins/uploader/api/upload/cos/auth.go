package cos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"hash"
	math_rand "math/rand"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	sha1SignAlgorithm   = "sha1"
	privateHeaderPrefix = "x-cos-"
	defaultAuthExpire   = time.Hour
)

func DNSScatterDialContextFunc(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	// DNS 打散
	math_rand.Seed(time.Now().UnixNano())
	start := math_rand.Intn(len(ips))
	for i := start; i < len(ips); i++ {
		conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ips[i].IP.String(), port))
		if err == nil {
			return
		}
	}
	for i := 0; i < start; i++ {
		conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ips[i].IP.String(), port))
		if err == nil {
			break
		}
	}
	return
}

// 需要校验的 Headers 列表
var NeedSignHeaders = map[string]bool{
	"host":                           true,
	"range":                          true,
	"x-cos-acl":                      true,
	"x-cos-grant-read":               true,
	"x-cos-grant-write":              true,
	"x-cos-grant-full-control":       true,
	"response-content-type":          true,
	"response-content-language":      true,
	"response-expires":               true,
	"response-cache-control":         true,
	"response-content-disposition":   true,
	"response-content-encoding":      true,
	"cache-control":                  true,
	"content-disposition":            true,
	"content-encoding":               true,
	"content-type":                   true,
	"content-length":                 true,
	"content-md5":                    true,
	"transfer-encoding":              true,
	"versionid":                      true,
	"expect":                         true,
	"expires":                        true,
	"x-cos-content-sha1":             true,
	"x-cos-storage-class":            true,
	"if-match":                       true,
	"if-modified-since":              true,
	"if-none-match":                  true,
	"if-unmodified-since":            true,
	"origin":                         true,
	"access-control-request-method":  true,
	"access-control-request-headers": true,
	"x-cos-object-type":              true,
}

func encodeURIComponent(s string, excluded ...[]byte) string {
	var b bytes.Buffer
	written := 0

	for i, n := 0, len(s); i < n; i++ {
		c := s[i]

		switch c {
		case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
			continue
		default:
			// Unreserved according to RFC 3986 sec 2.3
			if 'a' <= c && c <= 'z' {

				continue

			}
			if 'A' <= c && c <= 'Z' {

				continue

			}
			if '0' <= c && c <= '9' {

				continue
			}
			if len(excluded) > 0 {
				conti := false
				for _, ch := range excluded[0] {
					if ch == c {
						conti = true
						break
					}
				}
				if conti {
					continue
				}
			}
		}

		b.WriteString(s[written:i])
		fmt.Fprintf(&b, "%%%02X", c)
		written = i + 1
	}

	if written == 0 {
		return s
	}
	b.WriteString(s[written:])
	return b.String()
}

// 非线程安全，只能在进程初始化（而不是Client初始化）时做设置
func SetNeedSignHeaders(key string, val bool) {
	NeedSignHeaders[key] = val
}

func safeURLEncode(s string) string {
	s = encodeURIComponent(s)
	s = strings.Replace(s, "!", "%21", -1)
	s = strings.Replace(s, "'", "%27", -1)
	s = strings.Replace(s, "(", "%28", -1)
	s = strings.Replace(s, ")", "%29", -1)
	s = strings.Replace(s, "*", "%2A", -1)
	return s
}

type valuesSignMap map[string][]string

func (vs valuesSignMap) Add(key, value string) {
	key = strings.ToLower(safeURLEncode(key))
	vs[key] = append(vs[key], value)
}

func (vs valuesSignMap) Encode() string {
	var keys []string
	for k := range vs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		items := vs[k]
		sort.Strings(items)
		for _, val := range items {
			pairs = append(
				pairs,
				fmt.Sprintf("%s=%s", k, safeURLEncode(val)))
		}
	}
	return strings.Join(pairs, "&")
}

// AuthTime 用于生成签名所需的 q-sign-time 和 q-key-time 相关参数
type AuthTime struct {
	SignStartTime time.Time
	SignEndTime   time.Time
}

// NewAuthTime 生成 AuthTime 的便捷函数
//
//	expire: 从现在开始多久过期.
func NewAuthTime(expire time.Duration) *AuthTime {
	startTime := time.Now()
	signEndTime := startTime.Add(expire)
	return &AuthTime{
		SignStartTime: startTime,
		SignEndTime:   signEndTime,
	}
}

// signString return q-sign-time string
func (a *AuthTime) signString() string {
	return fmt.Sprintf("%d;%d", a.SignStartTime.Unix(), a.SignEndTime.Unix())
}

// keyString return q-key-time string
func (a *AuthTime) keyString() string {
	return fmt.Sprintf("%d;%d", a.SignStartTime.Unix(), a.SignEndTime.Unix())
}

// newAuthorization 通过一系列步骤生成最终需要的 Authorization 字符串
func newAuthorization(secretID, secretKey string, req *http.Request, authTime *AuthTime, signHost bool) string {
	signTime := authTime.signString()
	keyTime := authTime.keyString()
	signKey := calSignKey(secretKey, keyTime)

	if signHost {
		req.Header.Set("Host", req.Host)
	}
	formatHeaders := *new(string)
	signedHeaderList := *new([]string)
	formatHeaders, signedHeaderList = genFormatHeaders(req.Header)
	formatParameters, signedParameterList := genFormatParameters(req.URL.Query())
	formatString := genFormatString(req.Method, *req.URL, formatParameters, formatHeaders)
	stringToSign := calStringToSign(sha1SignAlgorithm, keyTime, formatString)
	signature := calSignature(signKey, stringToSign)

	return genAuthorization(
		secretID, signTime, keyTime, signature, signedHeaderList,
		signedParameterList,
	)
}

// AddAuthorizationHeader 给 req 增加签名信息
func AddAuthorizationHeader(req *http.Request, authTime *AuthTime) {
	if req.Header.Get("SecretID") == "" {
		return
	}

	auth := newAuthorization(req.Header.Get("SecretID"), req.Header.Get("SecretKey"), req,
		authTime, true,
	)
	req.Header.Set("Authorization", auth)
	req.Header.Del("SecretID")
	req.Header.Del("SecretKey")
}

func GetAuthorization(secretID, secretKey string, req *http.Request, authTime *AuthTime) string {
	if secretID == "" {
		return ""
	}

	auth := newAuthorization(secretID, secretKey, req,
		authTime, true,
	)
	return auth
}

// calSignKey 计算 SignKey
func calSignKey(secretKey, keyTime string) string {
	digest := calHMACDigest(secretKey, keyTime, sha1SignAlgorithm)
	return fmt.Sprintf("%x", digest)
}

// calStringToSign 计算 StringToSign
func calStringToSign(signAlgorithm, signTime, formatString string) string {
	h := sha1.New()
	h.Write([]byte(formatString))
	return fmt.Sprintf("%s\n%s\n%x\n", signAlgorithm, signTime, h.Sum(nil))
}

// calSignature 计算 Signature
func calSignature(signKey, stringToSign string) string {
	digest := calHMACDigest(signKey, stringToSign, sha1SignAlgorithm)
	return fmt.Sprintf("%x", digest)
}

// genAuthorization 生成 Authorization
func genAuthorization(secretID, signTime, keyTime, signature string, signedHeaderList, signedParameterList []string) string {
	return strings.Join([]string{
		"q-sign-algorithm=" + sha1SignAlgorithm,
		"q-ak=" + secretID,
		"q-sign-time=" + signTime,
		"q-key-time=" + keyTime,
		"q-header-list=" + strings.Join(signedHeaderList, ";"),
		"q-url-param-list=" + strings.Join(signedParameterList, ";"),
		"q-signature=" + signature,
	}, "&")
}

// genFormatString 生成 FormatString
func genFormatString(method string, uri url.URL, formatParameters, formatHeaders string) string {
	formatMethod := strings.ToLower(method)
	formatURI := uri.Path
	return fmt.Sprintf("%s\n%s\n%s\n%s\n", formatMethod, formatURI,
		formatParameters, formatHeaders,
	)
}

// genFormatParameters 生成 FormatParameters 和 SignedParameterList
// instead of the url.Values{}
func genFormatParameters(parameters url.Values) (formatParameters string, signedParameterList []string) {
	ps := valuesSignMap{}
	for key, values := range parameters {
		for _, value := range values {
			ps.Add(key, value)
			signedParameterList = append(signedParameterList, strings.ToLower(safeURLEncode(key)))
		}
	}
	//formatParameters = strings.ToLower(ps.Encode())
	formatParameters = ps.Encode()
	sort.Strings(signedParameterList)
	return
}

// genFormatHeaders 生成 FormatHeaders 和 SignedHeaderList
func genFormatHeaders(headers http.Header) (formatHeaders string, signedHeaderList []string) {
	hs := valuesSignMap{}
	for key, values := range headers {
		if isSignHeader(strings.ToLower(key)) {
			for _, value := range values {
				hs.Add(key, value)
				signedHeaderList = append(signedHeaderList, strings.ToLower(safeURLEncode(key)))
			}
		}
	}
	formatHeaders = hs.Encode()
	sort.Strings(signedHeaderList)
	return
}

// HMAC 签名
func calHMACDigest(key, msg, signMethod string) []byte {
	var hashFunc func() hash.Hash
	switch signMethod {
	case "sha1":
		hashFunc = sha1.New
	default:
		hashFunc = sha1.New
	}
	h := hmac.New(hashFunc, []byte(key))
	h.Write([]byte(msg))
	return h.Sum(nil)
}

func isSignHeader(key string) bool {
	for k, v := range NeedSignHeaders {
		if key == k && v {
			return true
		}
	}
	return strings.HasPrefix(key, privateHeaderPrefix)
}
