package main

import "flag"

type (
	config struct {
		grpcPort string
	}
)

func setConfig() *config {
	var cfg config
	flag.StringVar(&cfg.grpcPort, "grpcPort", ":8080", "")
	return &cfg
}
