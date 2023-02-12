package main

import (
	"log"
	"os"

	"github.com/manics/binderhub-container-registry-helper/amazon"
	"github.com/manics/binderhub-container-registry-helper/common"
)

var (
	// Version is set at build time using the Git repository metadata
	Version string
)

// The main entrypoint for the service
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	versionInfo := map[string]string{
		"version": Version,
	}

	registryH, err := amazon.Setup(os.Args[1:])
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	listen := "0.0.0.0:8080"
	common.Run(registryH, versionInfo, listen)
}
