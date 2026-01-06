package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/simjinhyun/x"
)

var (
	Version  string
	Revision string
	Date     string
	Go       string
)

var Build = struct {
	Version   string
	Revision  string
	BuildDate string
	Go        string
}{
	Version:   Version,
	Revision:  Revision,
	BuildDate: Date,
	Go:        Go,
}

func main() {
	a := x.NewApp()
	a.Logger.SetTimezone("Asia/Seoul")
	a.Logger.SetLevel(x.LevelDebug)

	a.Logger.Info("Go Version", Build.Go)
	a.Logger.Info("Version", Build.Version)
	a.Logger.Info("Revision", Build.Revision)
	a.Logger.Info("BuildDate", Build.BuildDate)

	a.AddConn(
		"db1",
		"mysql",
		"root:Tldrmf#2013@tcp(10.0.0.200:3306)/testdb?timeout=5s&readTimeout=30s&writeTimeout=30s",
	)

	a.Router.AddPreprocessors(PP1, PP2, PP3)
	a.Router.AddRoute(a, "POST", "/hello", x.ReplyJSON, MDW1, MDW2, MDW3, About)

	a.Run("localhost:7000", 5)
}

func PP1(c *x.Context) {}
func PP2(c *x.Context) {}
func PP3(c *x.Context) {}

func MDW1(c *x.Context) {}
func MDW2(c *x.Context) {}
func MDW3(c *x.Context) {}
func MDW4(c *x.Context) {}
func MDW5(c *x.Context) {}

func About(c *x.Context) {
	c.Response.Data = &Build
}
