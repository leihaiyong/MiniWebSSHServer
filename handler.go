package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

func listTermHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "index.template", termStore.All())
}

type newTermReq struct {
	Host     string `query:"host" form:"host"`
	Port     int    `query:"port" form:"port"`
	User     string `query:"user" form:"user"`
	Password string `query:"pwd" form:"pwd"`
	Rows     int    `query:"rows" form:"rows"`
	Cols     int    `query:"cols" form:"cols"`
}

func newTermHandler(c echo.Context) error {
	req := new(newTermReq)

	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Host == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			"Host not provided")
	}

	if req.Port == 0 {
		req.Port = 22
	}

	if req.User == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			"User or password not provided")
	}

	link := &TermLink{
		Host: req.Host,
		Port: req.Port,
	}

	err := link.Dial(req.User, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Rows == 0 || req.Cols == 0 {
		req.Rows = 40
		req.Cols = 80
	}

	term, err := link.NewTerm(req.Rows, req.Cols)
	if err != nil {
		link.Close()
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	c.Logger().Infof("Created term: %s", term)

	termStore.Put(term)

	return c.Render(http.StatusOK, "term.template", term)
}

type connTermReq struct {
	TermId string `query:"term"`
}

const TermBufferSize = 8192

func connTermHandler(c echo.Context) error {
	req := new(connTermReq)

	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	term := termStore.Get(req.TermId)
	if term == nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			"Term not exist: "+req.TermId)
	}

	websocket.Handler(func(ws *websocket.Conn) {
		defer func() {
			c.Logger().Infof("Destroy term: %s", term)
			term.Close()
			termStore.Remove(term)
			ws.Close()
		}()

		c.Logger().Infof("Linking term: %s", term)

		go func() {
			b := [TermBufferSize]byte{}
			for {
				n, err := term.Stdout.Read(b[:])
				if err != nil {
					if !errors.Is(err, io.EOF) {
						websocket.Message.Send(ws,
							fmt.Sprintf("\nError: %s", err.Error()))
						c.Logger().Error(err)
					}
					return
				}
				if n == 0 {
					continue
				}
				websocket.Message.Send(ws, string(b[:n]))
			}
		}()

		go func() {
			b := [TermBufferSize]byte{}
			for {
				n, err := term.Stderr.Read(b[:])
				if err != nil {
					if !errors.Is(err, io.EOF) {
						websocket.Message.Send(ws,
							fmt.Sprintf("\nError: %s", err.Error()))
						c.Logger().Error(err)
					}
					return
				}
				if n == 0 {
					continue
				}
				websocket.Message.Send(ws, string(b[:n]))
			}
		}()

		for {
			b := ""
			err := websocket.Message.Receive(ws, &b)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					c.Logger().Error(err)
				}
				return
			}
			_, err = term.Stdin.Write([]byte(b))
			if err != nil {
				if !errors.Is(err, io.EOF) {
					websocket.Message.Send(ws,
						fmt.Sprintf("\nError: %s", err.Error()))
					c.Logger().Error(err)
				}
				return
			}
		}
	}).ServeHTTP(c.Response(), c.Request())

	return nil
}
