package service

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	shadybusinessbot "github.com/Lawliet18/shady-business-bot"
	"github.com/Lawliet18/shady-business-bot/internal/message"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/ziflex/lecho/v3"
)

type Service struct {
	addr    string
	log     zerolog.Logger
	msgChan chan<- message.Message
	webFS   embed.FS
}

func New(
	log zerolog.Logger,
	addr string,
	msgChan chan<- message.Message,
) *Service {
	return &Service{
		log:     log.With().Str("component", "service").Logger(),
		addr:    addr,
		msgChan: msgChan,
		webFS:   shadybusinessbot.WebFS,
	}
}

type requestArgs struct {
	Name  string `param:"name" query:"name" form:"name" json:"name" xml:"name"`
	Phone string `param:"phone" query:"phone" form:"phone" json:"phone" xml:"phone"`
}

func (svc *Service) Start(ctx context.Context) error {
	e := echo.New()

	{
		logger := lecho.From(svc.log)
		e.Use(middleware.RequestID())
		e.Use(lecho.Middleware(lecho.Config{
			Logger:      logger,
			HandleError: true,
			Skipper: func(c echo.Context) bool {
				return strings.HasPrefix(c.Request().URL.Path, "/static") ||
					strings.HasPrefix(c.Request().URL.Path, "/favicon.ico")
			},
		}))
		e.Logger = logger
	}

	staticFS, err := fs.Sub(svc.webFS, "web")
	if err != nil {
		return fmt.Errorf("sub: %w", err)
	}

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, "/index.html")
	})
	e.StaticFS("/", staticFS)

	e.Any("/api", func(c echo.Context) error {
		var args requestArgs
		err := c.Bind(&args)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "cannot bind args", err)
		}

		args.Name = strings.TrimSpace(args.Name)
		args.Phone = strings.TrimSpace(args.Phone)

		if args.Name == "" || args.Phone == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "phone or name must not be empty")
		}

		msg := message.Message{
			Name:  args.Name,
			Phone: args.Phone,
		}

		select {
		case svc.msgChan <- msg:
			return c.Redirect(http.StatusTemporaryRedirect, "/congrats.html")
		case <-ctx.Done():
			return echo.NewHTTPError(http.StatusServiceUnavailable, "context cancelled")
		}
	})

	go func() {
		<-ctx.Done()

		cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := e.Shutdown(cancelCtx)
		if err != nil {
			svc.log.Err(err).Msg("shutdown server")
		}
	}()

	err = e.Start(svc.addr)
	if err != nil {
		switch {
		case errors.Is(err, http.ErrServerClosed):
			return nil
		default:
			return fmt.Errorf("run http server: %w", err)
		}
	}

	return nil
}
