package main

import (
	"fmt"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "NNNN"
	}

	fmt.Printf("Server started in port %s\n", port)

}
