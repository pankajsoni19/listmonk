package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

type serverConfig struct {
	RootURL       string                   `json:"root_url"`
	Messengers    []map[string]interface{} `json:"messengers"`
	Langs         []i18nLang               `json:"langs"`
	Lang          string                   `json:"lang"`
	Permissions   json.RawMessage          `json:"permissions"`
	Update        *AppUpdate               `json:"update"`
	NeedsRestart  bool                     `json:"needs_restart"`
	HasLegacyUser bool                     `json:"has_legacy_user"`
	Version       string                   `json:"version"`
}

// handleGetServerConfig returns general server config.
func handleGetServerConfig(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	out := serverConfig{
		RootURL:       app.constants.RootURL,
		Lang:          app.constants.Lang,
		Permissions:   app.constants.PermissionsRaw,
		HasLegacyUser: app.constants.HasLegacyUser,
	}

	// Language list.
	langList, err := getI18nLangList(app.constants.Lang, app)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading language list: %v", err))
	}
	out.Langs = langList

	// Sort messenger names with `email` always as the first item.
	messengers := make([]map[string]interface{}, 0)
	for _, v := range app.messengers {
		messengers = append(messengers, map[string]interface{}{
			"name": v.Name(),
			"uuid": v.UUID(),
		})
	}

	out.Messengers = messengers

	app.Lock()
	out.NeedsRestart = app.needsRestart
	out.Update = app.update
	app.Unlock()
	out.Version = versionString

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCharts returns chart data points to render ont he dashboard.
func handleGetDashboardCharts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	out, err := app.core.GetDashboardCharts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCounts returns stats counts to show on the dashboard.
func handleGetDashboardCounts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	out, err := app.core.GetDashboardCounts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleReloadApp restarts the app.
func handleReloadApp(c echo.Context) error {
	app := c.Get("app").(*App)
	go func() {
		<-time.After(time.Millisecond * 500)
		app.chReload <- syscall.SIGHUP
	}()
	return c.JSON(http.StatusOK, okResp{true})
}
