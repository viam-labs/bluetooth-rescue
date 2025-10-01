package main

import (
	"bluetoothrescue"
	"context"
	"flag"
	"sync"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils"
)

var mode = flag.String("mode", "", "pass wait, wait-rescue, or rescue to run scenarios without a viam-server parent")

func testMode(mode string) {
	logger := logging.NewLogger("rescue-test")
	logger.Info("starting test mode")
	var wg sync.WaitGroup
	ch := make(chan bluetoothrescue.DmesgLine)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if mode == "wait" || mode == "wait-rescue" {
		// start reader routine
		wg.Add(1)
		utils.PanicCapturingGo(func() {
			logger.Info("starting DmesgReader")
			err := bluetoothrescue.DmesgReader(ctx, ch)
			if err != nil {
				logger.Errorf("DmesgReader failed with %s", err)
			}
			wg.Done()
		})
	}

	if mode == "wait-rescue" {
		// start rescuer routine
		logger.Info("starting RescueRoutine")
		wg.Add(1)
		utils.PanicCapturingGo(func() {
			bluetoothrescue.RescueRoutine(ctx, ch, logger, &wg)
		})
	} else if mode == "rescue" {
		logger.Info("rescuing immediately")
		bluetoothrescue.RestartBluetooth(ctx, logger)
	}
	logger.Info("waiting for background tasks")
	wg.Wait()
}

func main() {
	flag.Parse()
	if *mode != "" {
		testMode(*mode)
		return
	}
	module.ModularMain(resource.APIModel{API: sensor.API, Model: bluetoothrescue.Rescue})
}
