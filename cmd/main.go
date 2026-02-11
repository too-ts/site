package main

import (
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/labstack/echo/v5/middleware"
	"github.com/labstack/gommon/log"
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

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

type State struct{}

func NewData() *State {
	return &State{}
}

func init() {
}

func main() {
	debug := true
	godotenv.Load()

	e := echo.New()

	if debug {
		e.Logger.SetLevel(log.DEBUG)
	}

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())
	e.Use(middleware.Secure())

	data := NewData()

	e.Renderer = newTemplate()
	e.Static("/css", "css")

	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index.html", data)
	})

	if debug {
		e.Logger.Fatal(e.Start(":42069"))
	} else {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("example.com", "www.example.com"),
			Cache: autocert.DirCache("/var/www/.cache"),
			Email:   "kareljf@gmail.com", 
		}
		sc := echo.StartConfig{
			Address:   ":8443",
			TLSConfig: m.TLSConfig(),
		}
		if err := sc.Start(context.Background(), e); err != nil {
			e.Logger.Error("failed to start server", "error", err)
		}

		err := e.StartTLS(":443", fullchain, privkey)
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}
