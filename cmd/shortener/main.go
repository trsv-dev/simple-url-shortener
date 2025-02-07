package main

import (
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/server"
)

func main() {
	if err := server.RunServer(); err != nil {
		panic(err)
	}
}
