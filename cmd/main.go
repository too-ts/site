package main

import (
	"context"
	"html/template"
	"io"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"golang.org/x/crypto/acme/autocert"
)

type Template struct {
	tmpl *template.Template
}

func newTemplate() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c *echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

type State struct{}

func NewData() *State {
	return &State{}
}

func init() {
}

func main() {
	godotenv.Load()

	e := echo.New()

	e.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.Secure())

	data := NewData()

	e.Static("/css", "css")

	e.GET("/", func(c *echo.Context) error {
		return c.Render(200, "index.html", data)
	})

	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("toots.dev", "www.toots.dev"),
		Cache:      autocert.DirCache("/var/www/.cache"),
		Email:      "kareljf@gmail.com",
	}
	sc := echo.StartConfig{
		Address:   ":8443",
		TLSConfig: m.TLSConfig(),
	}
	if err := sc.Start(context.Background(), e); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
