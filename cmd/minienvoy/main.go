package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"minienvoy/internal/admin"
	"minienvoy/internal/app"
	"minienvoy/internal/config"
	"minienvoy/internal/logging"
	"minienvoy/internal/probe"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	cfgPath := flag.String("config", envOr("MINIENVY_CONFIG", "configs/demo.yaml"), "config yaml")
	flag.Parse()

	log := logging.New(envOr("MINIENVY_LOG_LEVEL", "info"))
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	if v := os.Getenv("MINIENVY_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
		log = logging.New(v)
	}
	if err := config.MustRemoteToken(cfg.Admin.Bind); err != nil {
		log.Error("admin bind", "err", err)
		os.Exit(1)
	}
	rt, err := app.Build(cfg, log)
	if err != nil {
		log.Error("runtime", "err", err)
		os.Exit(1)
	}
	rt.Start()
	stopProbe := make(chan struct{})
	go probe.Loop(stopProbe, rt.Ups.All(), cfg.HealthEvery(), probe.Config{
		Timeout:       time.Duration(cfg.Health.TimeoutMS) * time.Millisecond,
		FailThreshold: uint64(cfg.Health.FailThreshold),
		PassThreshold: uint64(cfg.Health.PassThreshold),
	})

	api := &admin.API{
		Token:     config.AdminToken(),
		Health:    rt.Health,
		Overview:  rt.Overview,
		Routes:    rt.RouteViews,
		Upstreams: rt.UpstreamViews,
		Topology:  rt.Topology,
		Eject:     rt.Eject,
		Restore:   rt.Restore,
	}
	srv, err := admin.Listen(cfg.Admin.Bind, api.Handler())
	if err != nil {
		log.Error("admin listen", "err", err)
		os.Exit(1)
	}
	log.Info("minienvoy started", "listen", fmt.Sprintf("%s:%d", cfg.Listen.IP, cfg.Listen.Port), "admin", cfg.Admin.Bind, "reactors", cfg.Reactors)

	stopSIGHUP := config.WatchSIGHUP(func() {
		ncfg, err := config.Load(*cfgPath)
		if err != nil {
			log.Error("reload rejected", "err", err)
			return
		}
		log.Info("config reloaded (listen/reactor count require restart)", "routes", len(ncfg.Routes))
	})

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	stopSIGHUP()
	close(stopProbe)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Idle())
	defer cancel()
	_ = srv.Shutdown(ctx)
	rt.Stop()
	log.Info("minienvoy stopped")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func healthcheck() int {
	bind := os.Getenv("MINIENVY_HEALTH_URL")
	if bind == "" {
		bind = "http://127.0.0.1:8080/healthz"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bind, nil)
	if err != nil {
		return 1
	}
	client := *http.DefaultClient
	client.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
		*next = *next.Clone(context.WithoutCancel(next.Context()))
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
