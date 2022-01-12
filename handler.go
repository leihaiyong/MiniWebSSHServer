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
	Host     string `query:"host" form:"host" json:"host"`
	Port     int    `query:"port" form:"port" json:"port"`
	User     string `query:"user" form:"user" json:"user"`
	Password string `query:"pwd" form:"pwd" json:"password"`
	Rows     int    `query:"rows" form:"rows" json:"rows"`
	Cols     int    `query:"cols" form:"cols" json:"cols"`
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

	termStore.Add(term)

	return c.Render(http.StatusOK, "term.template", term)
}

type termErr struct {
	Cause string `json:"cause"`
}

func createTermHandler(c echo.Context) error {
	req := new(newTermReq)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	if req.Host == "" {
		return c.JSON(http.StatusBadRequest, termErr{"Host not provided"})
	}

	if req.Port == 0 {
		req.Port = 22
	}

	if req.User == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, termErr{"User or password not provided"})
	}

	link := &TermLink{
		Host: req.Host,
		Port: req.Port,
	}

	err := link.Dial(req.User, req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	if req.Rows == 0 || req.Cols == 0 {
		req.Rows = 40
		req.Cols = 80
	}

	term, err := link.NewTerm(req.Rows, req.Cols)
	if err != nil {
		link.Close()
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	c.Logger().Infof("Created term: %s", term)

	termStore.Add(term)

	return c.JSON(http.StatusOK, term)
}

type setTermWindowSizeReq struct {
	Rows int `query:"rows" form:"rows" json:"rows"`
	Cols int `query:"cols" form:"cols" json:"cols"`
}

func setTermWindowSizeHandler(c echo.Context) error {
	req := new(setTermWindowSizeReq)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	if req.Rows == 0 || req.Cols == 0 {
		return c.JSON(http.StatusBadRequest, termErr{"Rows or cols can't be zero"})
	}

	termId := c.Param("id")
	term, err := termStore.Do(termId, func(term *Term) error {
		return term.SetWindowSize(req.Rows, req.Cols)
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	return c.JSON(http.StatusOK, term)
}

const TermBufferSize = 8192

func linkTermDataHandler(c echo.Context) error {
	termId := c.Param("id")
	term, err := termStore.Get(termId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, termErr{err.Error()})
	}

	websocket.Handler(func(ws *websocket.Conn) {
		defer func() {
			c.Logger().Infof("Destroy term: %s", term)
			termStore.Del(term, true)
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
