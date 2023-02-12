// Based on https://golang.cafe/blog/golang-rest-api-example.html

package main

import (
	"log"
	"os"

	"github.com/manics/binderhub-container-registry-helper/amazon"
	"github.com/manics/binderhub-container-registry-helper/oracle"
	"github.com/manics/binderhub-container-registry-helper/registry"
)

var (
	// Version is set at build time using the Git repository metadata
	Version string
)

// The main entrypoint for the service
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s [amazon|oracle] ...\n", os.Args[0])
	}

	versionInfo := map[string]string{
		"version": Version,
	}
	var registryH registry.IRegistryClient
	var err error

	provider := os.Args[1]
	switch provider {
	case "amazon":
		registryH, err = amazon.Setup(os.Args[2:])
	case "oracle":
		registryH, err = oracle.Setup(os.Args[2:])
	default:
		log.Fatalf("Unknown provider: %s\n", provider)
	}
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	listen := "0.0.0.0:8080"
	registry.Run(registryH, &versionInfo, listen)
}
