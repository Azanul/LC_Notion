package main

import (
	"github.com/Azanul/lcnotion/api"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("test.env")
	api.Integrator()
}
