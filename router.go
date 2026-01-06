package x

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

type Router struct {
	WebRoot           string
	trees             map[string]*node
	preprocessors     []HandlerFunc
	preprocessorNames []string
}

type node struct {
	segment  string
	children map[string]*node
	route    *Route
}

// 여러 전처리기를 한 번에 추가
func (r *Router) AddPreprocessors(hs ...HandlerFunc) {
	for _, h := range hs {
		r.preprocessors = append(r.preprocessors, h)
		name := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
		r.preprocessorNames = append(r.preprocessorNames, name)
	}
}

func NewRouter() *Router {
	// 실행 중인 바이너리의 경로 (심볼릭 링크일 수 있음)
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// 심볼릭 링크를 실제 경로로 해석
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		panic(err)
	}

	// 절대 경로로 정규화
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		panic(err)
	}

	dir := filepath.Dir(exePath)
	WebRoot := filepath.Join(dir, "www")

	return &Router{
		WebRoot: WebRoot,
		trees:   make(map[string]*node),
	}
}

func (r *Router) CreateIndexFiles() {
	if err := os.MkdirAll(r.WebRoot, 0755); err != nil {
		panic(err)
	}
	filepath.WalkDir(r.WebRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}

		if d.IsDir() {
			index := filepath.Join(path, "index.html")
			_, statErr := os.Stat(index)

			if os.IsNotExist(statErr) {
				writeErr := os.WriteFile(index, []byte{}, 0644)
				if writeErr != nil {
					panic(writeErr)
				}
			} else if statErr != nil {
				panic(statErr)
			}
		}
		return nil
	})
}

type HandlerFunc func(*Context)

type Route struct {
	Path         string
	Method       string
	Reply        HandlerFunc
	Handlers     []HandlerFunc
	HandlerNames []string
	App          *App
}

func (r *Router) AddRoute(app *App, method, path string, reply HandlerFunc, handlers ...HandlerFunc) {
	// 루트 노드 준비
	if r.trees[method] == nil {
		r.trees[method] = &node{children: make(map[string]*node)}
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	cur := r.trees[method]

	for _, p := range parts {
		if cur.children[p] == nil {
			cur.children[p] = &node{segment: p, children: make(map[string]*node)}
		}
		cur = cur.children[p]
	}

	names := make([]string, len(handlers))
	for i, h := range handlers {
		names[i] = runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}

	cur.route = &Route{
		Path:         path,
		Method:       method,
		Reply:        reply,
		Handlers:     handlers,
		HandlerNames: names,
		App:          app,
	}
}

func (r *Router) findRoute(method, path string) *Route {
	root := r.trees[method]
	if root == nil {
		return nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	cur := root

	for _, p := range parts {
		next, ok := cur.children[p]
		if !ok {
			return nil
		}
		cur = next
	}

	return cur.route
}

func (r *Router) ServeHTTP(c *Context) {
	// 글로벌 전처리기 실행
	for i, pre := range r.preprocessors {
		c.Executed = append(c.Executed, r.preprocessorNames[i])
		pre(c)
	}

	route := r.findRoute(c.Req.Method, c.Req.URL.Path)
	if route != nil {
		for i, h := range route.Handlers {
			c.Executed = append(c.Executed, route.HandlerNames[i])
			c.Route = route
			h(c)
		}
		return
	}

	// 등록된 라우트가 없으면 정적 파일 제공
	path := filepath.Join(r.WebRoot, c.Req.URL.Path)
	http.ServeFile(c.Res, c.Req, path)
}
