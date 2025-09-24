package bluetoothrescue

import (
	"context"
	"errors"
	"sync"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils"
	"go.viam.com/utils/rpc"
)

var (
	Rescue           = resource.NewModel("viam", "bluetooth-rescue", "rescue")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterComponent(sensor.API, Rescue,
		resource.Registration[sensor.Sensor, *Config]{
			Constructor: newBluetoothRescueRescue,
		},
	)
}

type Config struct {
	// when false, this logs hardware errors but doesn't try to fix them.
	Rescue bool `json:"rescue"`
}

// Validate ensures all parts of the config are valid and important fields exist.
// Returns implicit required (first return) and optional (second return) dependencies based on the config.
// The path is the JSON path in your robot's config (not the `Config` struct) to the
// resource being validated; e.g. "components.0".
func (cfg *Config) Validate(path string) ([]string, []string, error) {
	// Add config validation code here
	return nil, nil, nil
}

type rescuer struct {
	resource.AlwaysRebuild

	name resource.Name

	logger logging.Logger
	cfg    *Config

	cancelCtx  context.Context
	cancelFunc func()
	wg         sync.WaitGroup
}

func newBluetoothRescueRescue(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}
	model, err := NewRescue(ctx, deps, rawConf.ResourceName(), conf, logger)
	if err != nil {
		return nil, err
	}
	rescuer := model.(*rescuer)

	// todo: how to make this not double-repair if restarted?
	// - make sure bluetooth is broken before repairing
	// todo: how to make this not flap / fight when someone disables bluetooth (i.e. maybe it should be off)

	// todo: callback and restart these
	ch := make(chan DmesgLine)

	// start reader routine
	rescuer.wg.Add(1)
	utils.PanicCapturingGo(func() {
		err := DmesgReader(rescuer.cancelCtx, ch)
		if err != nil {
			logger.Errorf("DmesgReader failed with %s", err)
		}
		rescuer.wg.Done()
	})

	// start rescuer routine
	// todo: reconfigure won't start/stop this; how to make that clear to the user?
	rescuer.wg.Add(1)
	utils.PanicCapturingGo(func() {
		for line := range ch {
			if line.Message == hardwareErrorMsg {
				rescuer.logger.Warnf("dmesg tailer found hardware error at %s", line.Timestamp)
				if err := rescuer.rescue(); err != nil {
					rescuer.logger.Errorf("rescue failed with %s", err)
					// todo: think about retry strategy
				}
			}
		}
		rescuer.wg.Done()
	})

	return model, nil
}

func NewRescue(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *Config, logger logging.Logger) (sensor.Sensor, error) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	s := &rescuer{
		name:       name,
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}
	return s, nil
}

func (s *rescuer) Name() resource.Name {
	return s.name
}

func (s *rescuer) NewClientFromConn(ctx context.Context, conn rpc.ClientConn, remoteName string, name resource.Name, logger logging.Logger) (sensor.Sensor, error) {
	panic("not implemented")
}

func (s *rescuer) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	panic("not implemented")
}

func (s *rescuer) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	panic("not implemented")
}

func (s *rescuer) Close(context.Context) error {
	s.cancelFunc()
	s.wg.Wait()
	return nil
}
