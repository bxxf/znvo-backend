package main

import (
	"context"

	"github.com/bxxf/znvo-backend/internal/server"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(
			server.NewServer,
		),
		fx.Invoke(
			func(s *server.Server) {
				s.ListenAndServe()
			},
		),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		panic(err)
	}

	<-app.Done()

	if err := app.Stop(ctx); err != nil {
		panic(err)
	}

}
