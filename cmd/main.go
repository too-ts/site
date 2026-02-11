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
	templates *template.Template
}

func (t *Template) Render(c *echo.Context, w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
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
	e.Use(middleware.CORS("https://www.toots.dev", "https://toots.dev"))
	e.Use(middleware.Secure())
	e.Use(middleware.RequestLogger())

	data := NewData()

	e.Static("/css", "css")

	e.GET("/", func(c *echo.Context) error {
		return c.Render(200, "index.html", data)
	})

	renderer := &Template{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}

	e.Renderer = renderer

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
