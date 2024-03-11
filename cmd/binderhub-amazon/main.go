package main

import (
	"log"
	"os"

	"github.com/manics/binderhub-container-registry-helper/amazon"
	"github.com/manics/binderhub-container-registry-helper/common"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
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

	// Custom Prometheus registry to disable default go metrics
	promRegistry := prometheus.NewRegistry()
	promRegistry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	registryH, err := amazon.Setup(promRegistry, os.Args[1:])
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	listen := "0.0.0.0:8080"
	common.Run(registryH, versionInfo, listen, promRegistry)
}
