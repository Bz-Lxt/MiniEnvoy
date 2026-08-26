package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type File struct {
	Version  int           `yaml:"version"`
	LogLevel string        `yaml:"log_level"`
	Reactors int           `yaml:"reactors"`
	Listen   Listen        `yaml:"listen"`
	Admin    Admin         `yaml:"admin"`
	Buffer   Buffer        `yaml:"buffer"`
	Timeouts Timeouts      `yaml:"timeouts"`
	Health   Health        `yaml:"health"`
	Routes   []Route       `yaml:"routes"`
	Upstreams []Upstream   `yaml:"upstreams"`
}

type Listen struct {
	IP      string `yaml:"ip"`
	Port    int    `yaml:"port"`
	Backlog int    `yaml:"backlog"`
}

type Admin struct {
	Bind string `yaml:"bind"`
}

type Buffer struct {
	ChunkSize  int    `yaml:"chunk_size"`
	HighWater  int    `yaml:"high_water"`
	LowWater   int    `yaml:"low_water"`
	MaxPayload uint32 `yaml:"max_payload"`
}

type Timeouts struct {
	IdleMS     int `yaml:"idle_ms"`
	UpstreamMS int `yaml:"upstream_ms"`
	DrainMS    int `yaml:"drain_ms"`
}

type Health struct {
	IntervalMS     int `yaml:"interval_ms"`
	TimeoutMS      int `yaml:"timeout_ms"`
	FailThreshold  int `yaml:"fail_threshold"`
	PassThreshold  int `yaml:"pass_threshold"`
}

type Route struct {
	ID        uint32   `yaml:"id"`
	Name      string   `yaml:"name"`
	Algorithm string   `yaml:"algorithm"`
	Upstreams []string `yaml:"upstreams"`
}

type Upstream struct {
	ID     string `yaml:"id"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Weight int    `yaml:"weight"`
}

func Load(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	f.applyDefaults()
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return &f, nil
}

func (f *File) applyDefaults() {
	if f.Version == 0 {
		f.Version = 1
	}
	if f.LogLevel == "" {
		f.LogLevel = "info"
	}
	if f.Reactors <= 0 {
		f.Reactors = 1
	}
	if f.Listen.IP == "" {
		f.Listen.IP = "0.0.0.0"
	}
	if f.Listen.Port == 0 {
		f.Listen.Port = 9000
	}
	if f.Listen.Backlog <= 0 {
		f.Listen.Backlog = 1024
	}
	if f.Admin.Bind == "" {
		f.Admin.Bind = "127.0.0.1:18080"
	}
	if f.Buffer.ChunkSize <= 0 {
		f.Buffer.ChunkSize = 65536
	}
	if f.Buffer.HighWater <= 0 {
		f.Buffer.HighWater = f.Buffer.ChunkSize * 3 / 4
	}
	if f.Buffer.LowWater <= 0 {
		f.Buffer.LowWater = f.Buffer.ChunkSize / 4
	}
	if f.Buffer.MaxPayload == 0 {
		f.Buffer.MaxPayload = 1 << 20
	}
	if f.Timeouts.IdleMS <= 0 {
		f.Timeouts.IdleMS = 60000
	}
	if f.Timeouts.UpstreamMS <= 0 {
		f.Timeouts.UpstreamMS = 5000
	}
	if f.Timeouts.DrainMS <= 0 {
		f.Timeouts.DrainMS = 3000
	}
	if f.Health.IntervalMS <= 0 {
		f.Health.IntervalMS = 2000
	}
	if f.Health.TimeoutMS <= 0 {
		f.Health.TimeoutMS = 800
	}
	if f.Health.FailThreshold <= 0 {
		f.Health.FailThreshold = 3
	}
	if f.Health.PassThreshold <= 0 {
		f.Health.PassThreshold = 2
	}
	for i := range f.Upstreams {
		if f.Upstreams[i].Weight <= 0 {
			f.Upstreams[i].Weight = 1
		}
	}
}

func (f *File) Idle() time.Duration      { return time.Duration(f.Timeouts.IdleMS) * time.Millisecond }
func (f *File) UpstreamTO() time.Duration {
	return time.Duration(f.Timeouts.UpstreamMS) * time.Millisecond
}
func (f *File) HealthEvery() time.Duration {
	return time.Duration(f.Health.IntervalMS) * time.Millisecond
}
