package bluetoothrescue

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"go.viam.com/rdk/logging"
)

func TestDmesgReader(t *testing.T) {
	// todo: this needs a timeout
	logger := logging.NewTestLogger(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ch := make(chan DmesgLine)

	helloMsg := fmt.Sprintf("hello %s", time.Now().String())
	kmsg, err := os.OpenFile("/dev/kmsg", os.O_RDWR, os.ModeAppend)
	if err != nil {
		t.Errorf("can't open kmsg")
	}
	defer kmsg.Close()
	// without the newline this won't appear in dmesg right away
	_, err = kmsg.WriteString(helloMsg + "\n")
	if err != nil {
		t.Error(err)
	}
	logger.Debugf("wrote to /dev/kmsg: %s", helloMsg)

	wg := sync.WaitGroup{}
	wg.Add(1)

	// reader routine
	go func() {
		for line := range ch {
			if line.Message == helloMsg {
				logger.Info("found expected string in dmesg output")
				break
			}
		}
		wg.Done()
		cancel()
	}()

	err = DmesgReader(ctx, ch)
	if err != nil {
		t.Error(err)
	}
	wg.Wait()
}

func TestRestartBluetooth(t *testing.T) {
	if err := RestartBluetooth(t.Context(), logging.NewTestLogger(t)); err != nil {
		t.Error(err)
	}
}
