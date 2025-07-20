package http

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo     *echo.Echo
	handlers *HTTPHandlers
	port     int
}

func NewServer(port int) *Server {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	handlers := NewHTTPHandlers()

	return &Server{
		echo:     e,
		handlers: handlers,
		port:     port,
	}
}

func (s *Server) SetupRoutes() {
	s.echo.GET("/health", s.handlers.Health)
	s.echo.GET("/echo", s.handlers.EchoGet)
	s.echo.POST("/echo", s.handlers.EchoPost)
	s.echo.GET("/delay/:seconds", s.handlers.Delay)
	s.echo.GET("/status/:code", s.handlers.Status)
}

func (s *Server) GetEcho() *echo.Echo {
	return s.echo
}

func (s *Server) Start() error {
	s.SetupRoutes()
	return s.echo.Start(fmt.Sprintf(":%d", s.port))
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}