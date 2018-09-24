package main

import (
	"os"
	"encoding/json"
)

type ServerConfig struct {
	ID string `json:"id"`
	Addr string `json:"address"`
}

type Config struct {
	Servers []ServerConfig `json:"servers"`
}

func LoadConfig(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	data := make([]byte, 1024)
	size, err := f.Read(data)
	if err != nil {
		panic(err)
	}

	data = data[:size]

	c := &Config{}
	if err := json.Unmarshal(data, c); err != nil {
		panic(err)
	}

	return c
}