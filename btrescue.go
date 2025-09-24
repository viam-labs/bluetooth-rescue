package bluetoothrescue

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"github.com/Wifx/gonetworkmanager"
	"github.com/go-viper/mapstructure/v2"
)

const hardwareErrorMsg = "Bluetooth: hci0: hardware error 0x00"
const kernelModule = "hci_uart"

func readConnections(nm gonetworkmanager.Settings) ([]*NMConnection, error) {
	connections, err := nm.ListConnections()
	if err != nil {
		return nil, err
	}

	parsedCons := make([]*NMConnection, 0, len(connections))

	for _, con := range connections {
		settings, err := con.GetSettings()
		if err != nil {
			return nil, err
		}
		parsed := &NMConnection{raw: con}
		err = mapstructure.Decode(settings, parsed)
		if err != nil {
			return nil, err
		}
		parsedCons = append(parsedCons, parsed)
	}
	return parsedCons, nil
}

func getFirst[T any](items []*T, predicate func(*T) bool) *T {
	for _, item := range items {
		if predicate(item) {
			return item
		}
	}
	return nil
}

// returns first connection from NetworkManager with type=bluetooth
func getBTConnection(nm gonetworkmanager.Settings) (*NMConnection, error) {
	cons, err := readConnections(nm)
	if err != nil {
		return nil, err
	}
	// todo: warn if there is more than one option
	con := getFirst(cons, func(c *NMConnection) bool { return c.Connection.Type == "bluetooth" })
	if con == nil {
		return nil, fmt.Errorf("no bluetooth connection in network manager. %d candidates: %+v", len(cons), cons)
	}
	return con, nil
}

// make sure bluetooth is in a bad state + that it should be enabled, then fix it
func (s *rescuer) rescue() error {
	if !s.cfg.Rescue {
		s.logger.Info("not rescuing because rescue=false in config")
		return nil
	}
	nmSettings, err := gonetworkmanager.NewSettings()
	if err != nil {
		return err
	}
	con, err := getBTConnection(nmSettings)
	if err != nil {
		return err
	}
	s.logger.Infof("rescuing connection %q: %+v", con.Connection.ID, con)

	output, err := exec.CommandContext(s.cancelCtx, "rmmod", kernelModule).CombinedOutput()
	if err != nil {
		if bytes.Contains(output, []byte("not currently loaded")) {
			s.logger.Debugf("ignoring 'not loaded' error %q", string(output))
		} else {
			return fmt.Errorf("rmmod failed, error %q output %q", err, string(output))
		}
	}
	output, err = exec.CommandContext(s.cancelCtx, "modprobe", kernelModule).CombinedOutput()
	if err != nil {
		return fmt.Errorf("modprobe failed, error %q output %q", err, string(output))
	}

	tries := 2
	for i := range tries {
		time.Sleep(time.Second * 3)

		output, err := exec.CommandContext(s.cancelCtx, "nmcli", "c", "up", con.Connection.ID).CombinedOutput()
		if err == nil {
			s.logger.Infof("successfully brought up connection %q", con.Connection.ID)
			return nil
		}
		s.logger.Warnf("failed attempt %d/%d to bring up connection %q: %q", i+1, tries, con.Connection.ID, string(output))
	}
	return fmt.Errorf("failed after %d attempts to bring up connection %q", tries, con.Connection.ID)
}

// todo: we must have wrappers for this somewhere right
type NMConnection struct {
	Connection NMInnerConnection
	raw        gonetworkmanager.Connection
}

type NMInnerConnection struct {
	ID   string
	Type string
	UUID string
}
