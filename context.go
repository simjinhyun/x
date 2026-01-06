package x

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/simjinhyun/x/util"
)

type TextBytes []byte

func (b TextBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(b))
}
func (b TextBytes) String() string {
	return string(b)
}

// Context : 요청/응답을 담는 컨텍스트
type Context struct {
	App       *App
	Req       *http.Request
	Res       http.ResponseWriter
	AppError  *AppError
	RouteType string
	Store     map[string]any //핸들러 체인들이 자유롭게 데이터 담을 수 있게
	ReqID     string
	ReqTime   time.Time
	ReqBody   TextBytes
	RemoteIP  string
	Route     *Route
	Executed  []string
	Response  struct {
		Code    string
		Message string
		Data    any
		Elapsed string
	}
}

func NewContext(a *App, w http.ResponseWriter, r *http.Request) *Context {
	now := time.Now()
	c := &Context{
		App:      a,
		Req:      r,
		Res:      w,
		Store:    map[string]any{},
		ReqID:    util.EncodeToBase62(uint64(now.UnixNano())),
		ReqTime:  now,
		RemoteIP: getClientIP(r),
	}
	if a.Logger.GetLevel() == "DEBUG" {
		c.CopyBody()
	}

	return c
}

func (c *Context) CopyBody() {
	if c.Req.Body == nil {
		c.ReqBody = nil
		return
	}

	ct := c.Req.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/") {
		c.ReqBody = nil
		return
	}

	// 최대 1MB까지만 읽기
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Req.Body, 1024*1024))
	if err != nil {
		c.ReqBody = nil
		return
	}

	// Body 복원
	c.Req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	c.ReqBody = bodyBytes
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ipStr := strings.TrimSpace(ips[0])
			if net.ParseIP(ipStr) != nil {
				return ipStr
			}
		}
	}

	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		if net.ParseIP(xRealIP) != nil {
			return xRealIP
		}
	}

	if cfConnectingIP := r.Header.Get("CF-Connecting-IP"); cfConnectingIP != "" {
		if net.ParseIP(cfConnectingIP) != nil {
			return cfConnectingIP
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

var noErr = NewAppError("OK", nil, nil)

func (c *Context) Recover() {
	if rec := recover(); rec != nil {
		var appErr *AppError
		switch e := rec.(type) {
		case *AppError:
			appErr = e
		case error:
			appErr = NewAppError("RuntimeError", e, nil)
			c.App.Logger.Error(fmt.Sprintf("%s", debug.Stack()))
		default:
			appErr = NewAppError("RuntimeError", fmt.Errorf("%v", rec), nil)
			c.App.Logger.Error(fmt.Sprintf("%s", debug.Stack()))
		}
		c.AppError = appErr
	} else {
		c.AppError = noErr
	}

	c.Response.Code = c.AppError.Code
	c.Response.Elapsed = time.Since(c.ReqTime).String()

	//정적파일 서빙은 ServeFile 함수가 직접 응답함.
	c.App.Logger.Debug(c.Route)
	if c.Route != nil {
		c.Route.Reply(c)
	}

	//디버그 로그 (운영 성능 영향 제로)
	c.App.Logger.Debug(
		c.ReqID, "DONE",
		"IP", c.RemoteIP,
		c.Req.Method, c.Req.URL.Path,
		c.Response.Code, c.AppError.Src, c.AppError.Err,
		"ReqBody", c.ReqBody.String(),
		"Executed", c.Executed,
		"Elapsed", c.Response.Elapsed,
	)
}

func ReplyJSON(c *Context) {
	c.App.Logger.Debug("ReplyJSON")
	c.Res.Header().Set("Content-Type", "application/json; charset=utf-8")

	c.Response.Message = "메세지조립할것"
	if err := json.NewEncoder(c.Res).Encode(c.Response); err != nil {
		http.Error(c.Res, err.Error(), http.StatusInternalServerError)
	}
}

func ReplyHTML(c *Context) {
	c.App.Logger.Debug("ReplyHTML")
	c.Res.Header().Set("Content-Type", "text/html; charset=utf-8")

	if c.Response.Code == "OK" {
		if html, ok := c.Response.Data.(string); ok {
			fmt.Fprint(c.Res, html)
		} else {
			//응답데이터가 html 텍스트가 아니므로 JSON 마샬 응답
			ReplyJSON(c)
		}
	} else {
		fmt.Fprintf(
			c.Res,
			"<html><body><h1>Error: %s</h1></body></html>",
			c.Response.Code,
		)
	}
}

// 값 저장
func (c *Context) Set(key string, value any) {
	c.Store[key] = value
}

// 범용 Get: 그냥 any 반환
func (c *Context) Get(key string) any {
	return c.Store[key]
}
