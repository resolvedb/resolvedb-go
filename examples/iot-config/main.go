// IoT config example - device configuration with BDT tokens.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/resolvedb/resolvedb-go"
	"github.com/resolvedb/resolvedb-go/security"
)

// DeviceConfig represents IoT device configuration.
type DeviceConfig struct {
	FirmwareVersion string  `json:"firmware_version"`
	ReportInterval  int     `json:"report_interval_sec"`
	SensorThreshold float64 `json:"sensor_threshold"`
	Enabled         bool    `json:"enabled"`
}

func main() {
	// Create client with namespace
	client, err := resolvedb.New(
		resolvedb.WithNamespace("iot-fleet"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Generate BDT for this device (in production, persist and rotate weekly)
	bdt, err := security.NewBDT()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Device BDT: %s\n\n", bdt.String())

	// Query config using BDT (anonymous device identity)
	var config DeviceConfig
	err = client.Get(ctx, "device-config", "sensor-v1", &config,
		resolvedb.WithBDT(bdt.String()),
	)
	if err != nil {
		log.Printf("Config fetch error: %v", err)
		// Use defaults on error
		config = DeviceConfig{
			FirmwareVersion: "1.0.0",
			ReportInterval:  60,
			SensorThreshold: 25.0,
			Enabled:         true,
		}
	}

	fmt.Println("Device Configuration:")
	fmt.Printf("  Firmware: %s\n", config.FirmwareVersion)
	fmt.Printf("  Report Interval: %d seconds\n", config.ReportInterval)
	fmt.Printf("  Sensor Threshold: %.1f\n", config.SensorThreshold)
	fmt.Printf("  Enabled: %v\n", config.Enabled)
}
