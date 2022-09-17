package main

import (
	update "github.com/Azanul/lcnotion/api"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("test.env")
	update.Integrator()
}
