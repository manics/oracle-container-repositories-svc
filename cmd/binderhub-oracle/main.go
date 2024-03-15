package main

import (
	"log"
	"os"

	"github.com/manics/binderhub-container-registry-helper/common"
	"github.com/manics/binderhub-container-registry-helper/oracle"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	// Version is set at build time using the Git repository metadata
	Version string
)

// The main entrypoint for the service
func run(args []string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	versionInfo := map[string]string{
		"version": Version,
	}

	// Custom Prometheus registry to disable default go metrics
	promRegistry := prometheus.NewRegistry()
	promRegistry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	registryH, err := oracle.Setup(promRegistry, args)
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	listen := "0.0.0.0:8080"
	common.Run(registryH, versionInfo, listen, promRegistry)
}

func main() {
	run(os.Args[1:])
}
