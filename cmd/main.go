package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"golang.org/x/crypto/acme/autocert"
)

const (
	siteTitle       = "toots.dev"
	siteDescription = "Why do they call it oven when you off in the cold food of out hot eat the food."
	certCacheDir    = "/app/.cache"

	// Report-only for now so a policy mistake can't break the page.
	contentSecurityPolicy = "default-src 'self'; base-uri 'self'; object-src 'none'; form-action 'self'; frame-ancestors 'none'; script-src 'self' https://unpkg.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(c *echo.Context, w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type ViewData struct {
	Title       string
	Description string
	Code        int
	Message     string
}

func NewData() *ViewData {
	return &ViewData{
		Title:       siteTitle,
		Description: siteDescription,
	}
}

func htmlErrorHandler(c *echo.Context, err error) {
	code := echo.StatusCode(err)
	if code == 0 {
		code = http.StatusInternalServerError
	}

	message := http.StatusText(code)
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) && httpErr.Message != "" {
		message = httpErr.Message
	}

	if c.Request().Method == http.MethodHead {
		if respErr := c.NoContent(code); respErr != nil {
			c.Logger().Error("failed to send error response", "error", respErr)
		}
		return
	}

	data := &ViewData{
		Title:       fmt.Sprintf("%d - %s", code, siteTitle),
		Description: siteDescription,
		Code:        code,
		Message:     message,
	}
	if respErr := c.Render(code, "error", data); respErr != nil {
		echo.DefaultHTTPErrorHandler(false)(c, err)
	}
}

func extraSecurityHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		c.Response().Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		return next(c)
	}
}

func staticCacheHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if strings.HasPrefix(c.Request().URL.Path, "/css/") {
			c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=3600")
		}
		return next(c)
	}
}

func main() {
	godotenv.Load()

	e := echo.New()
	e.HTTPErrorHandler = htmlErrorHandler

	// The site sits behind Docker's port mapping, not a trusted reverse proxy,
	// so only the direct peer address is trustworthy.
	e.IPExtractor = echo.ExtractIPDirect()

	e.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	secureConfig := middleware.DefaultSecureConfig
	secureConfig.HSTSMaxAge = 31536000
	secureConfig.ReferrerPolicy = "strict-origin-when-cross-origin"
	secureConfig.ContentSecurityPolicy = contentSecurityPolicy
	secureConfig.CSPReportOnly = true

	e.Use(middleware.Recover())
	e.Use(extraSecurityHeaders)
	e.Use(middleware.SecureWithConfig(secureConfig))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{MinLength: 1024}))
	e.Use(staticCacheHeaders)
	e.Use(middleware.CORS("https://www.toots.dev", "https://toots.dev"))
	e.Use(middleware.RequestLogger())

	data := NewData()

	e.Static("/css", "css")

	e.GET("/robots.txt", func(c *echo.Context) error {
		return c.File("static/robots.txt")
	})
	e.GET("/sitemap.xml", func(c *echo.Context) error {
		return c.File("static/sitemap.xml")
	})

	e.GET("/", func(c *echo.Context) error {
		return c.Render(http.StatusOK, "index.html", data)
	})

	renderer := &Template{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}

	e.Renderer = renderer

	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("toots.dev", "www.toots.dev"),
		Cache:      autocert.DirCache(certCacheDir),
		Email:      "kareljf@gmail.com",
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{
		Address:         ":8443",
		TLSConfig:       m.TLSConfig(),
		GracefulTimeout: 10 * time.Second,
		OnShutdownError: func(err error) {
			e.Logger.Error("graceful shutdown failed", "error", err)
		},
		BeforeServeFunc: func(s *http.Server) error {
			s.ReadHeaderTimeout = 10 * time.Second
			s.ReadTimeout = 30 * time.Second
			s.WriteTimeout = 60 * time.Second
			s.IdleTimeout = 120 * time.Second
			s.MaxHeaderBytes = 1 << 20
			return nil
		},
	}
	if err := sc.Start(ctx, e); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
