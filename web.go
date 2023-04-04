package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type WebServer struct {
	*Endhouse
	*echo.Echo
}

func NewWebServer(h *Endhouse) *WebServer {
	s := WebServer{
		Endhouse: h,
		Echo:     echo.New(),
	}

	return &s
}

func (s *WebServer) SetupRoute() {
	s.Echo.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Up!")
	})

	s.Echo.POST("/task/:name", func(c echo.Context) error {
		if e := s.Endhouse.ExecuteTaskByName(c.Param("name")); e != nil {
			return e
		}
		return c.String(http.StatusOK, "Ok")
	})

}

func (s *WebServer) Run() {
	s.Echo.Logger.Fatal(s.Echo.Start(":1323"))
}
