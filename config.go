package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config map[string][]*Metric

type Metric struct {
	Name   string
	Match  string
	Value  string
	Labels map[string]string
}

func loadConfig(filename, namespace string) (Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := Config{}
	err = yaml.NewDecoder(f).Decode(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
