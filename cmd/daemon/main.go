// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/PandasWhoCode/pr-mirror-app/internal/api"
	"github.com/PandasWhoCode/pr-mirror-app/internal/application"
	"github.com/PandasWhoCode/pr-mirror-app/internal/logging"
	"github.com/PandasWhoCode/pr-mirror-app/internal/router"
	_ "github.com/joomcode/errorx"
	_ "github.com/labstack/echo/v5"
)

func main() {
	cfg := application.DefaultConfig()
	app := application.NewApplication(cfg)

	// Wire the application's health registry into the router config so the
	// v1 health/liveness/readiness handlers can snapshot per-component
	// status on every request. The registry is populated by Application
	// during Initialize() (lifecycle, http, https, database — only the
	// active subsystems are registered).
	routerCfg := router.NewConfig()
	routerCfg.HealthRegistry = app.HealthRegistry()

	_ = app.RegisterModule(api.Module(routerCfg))

	if err := app.Configure(); err != nil {
		logging.Daemon.
			Fatal().
			Err(err).
			Msgf("an unhandled error occurred during %s configuration", app.Name)
		os.Exit(1)
	}

	if err := app.Initialize(); err != nil {
		logging.Daemon.
			Fatal().
			Err(err).
			Msgf("an unhandled error occurred during %s initialization", app.Name)
		os.Exit(1)
	}

	ec, err := app.Start()

	if err != nil {
		logging.Daemon.
			Fatal().
			Err(err).
			Msgf("an unhandled error occurred causing %s to terminate unexpectedly", app.Name)
		os.Exit(ec)
	}
}
