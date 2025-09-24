package main

import (
	"bluetoothrescue"
	"fmt"
	"runtime/debug"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
)

func main() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		// todo: move this print to ModularMain
		fmt.Printf("version: %s, checksum: %s, go version: %s\n", info.Main.Version, info.Main.Sum, info.GoVersion)
	}
	module.ModularMain(resource.APIModel{sensor.API, bluetoothrescue.Rescue})
}
