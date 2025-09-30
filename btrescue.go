package bluetoothrescue

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"time"

	"github.com/Wifx/gonetworkmanager"
	"github.com/go-viper/mapstructure/v2"
	"go.viam.com/rdk/logging"
)

const hardwareErrorMsg = "Bluetooth: hci0: hardware error 0x00"
const kernelModule = "hci_uart"

// query NetworkManager, return a list of parsed connections
func readConnections(nm gonetworkmanager.Settings) ([]*NMConnection, error) {
	connections, err := nm.ListConnections()
	if err != nil {
		return nil, err
	}
	mgr, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return nil, err
	}
	actives, err := mgr.GetPropertyActiveConnections()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(actives))
	for _, active := range actives {
		id, err := active.GetPropertyUUID()
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	parsedCons := make([]*NMConnection, 0, len(connections))

	for _, con := range connections {
		settings, err := con.GetSettings()
		if err != nil {
			return nil, err
		}
		parsed := &NMConnection{}
		err = mapstructure.Decode(settings, parsed)
		if err != nil {
			return nil, err
		}
		parsed.raw = con
		parsed.active = slices.ContainsFunc(ids, func(id string) bool {
			return id == parsed.Connection.UUID
		})
		parsedCons = append(parsedCons, parsed)
	}
	return parsedCons, nil
}

// return first item in slice that has predicate(x)=true, else nil
func getFirst[T any](items []*T, predicate func(*T) bool) *T {
	i := slices.IndexFunc(items, predicate)
	if i == -1 {
		return nil
	}
	return items[i]
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
func RestartBluetooth(ctx context.Context, logger logging.Logger) error {
	nmSettings, err := gonetworkmanager.NewSettings()
	if err != nil {
		return err
	}
	con, err := getBTConnection(nmSettings)
	if err != nil {
		return err
	}
	logger.Infof("found NetworkManager connection %q: %+v", con.Connection.ID, con)
	if con.active {
		logger.Warn("not rescuing connection because NetworkManager considers it active")
		return nil
	}

	output, err := exec.CommandContext(ctx, "rmmod", kernelModule).CombinedOutput()
	if err != nil {
		if bytes.Contains(output, []byte("not currently loaded")) {
			logger.Debugf("ignoring 'not loaded' error %q", string(output))
		} else {
			return fmt.Errorf("rmmod failed, error %q output %q", err, string(output))
		}
	}
	output, err = exec.CommandContext(ctx, "modprobe", kernelModule).CombinedOutput()
	if err != nil {
		return fmt.Errorf("modprobe failed, error %q output %q", err, string(output))
	}
	logger.Info("restarted kernel module")

	tries := 2
	for i := range tries {
		time.Sleep(time.Second * 5)

		output, err := exec.CommandContext(ctx, "nmcli", "c", "up", con.Connection.ID).CombinedOutput()
		if err == nil {
			logger.Infof("successfully brought up connection %q", con.Connection.ID)
			return nil
		}
		logger.Warnf("failed attempt %d/%d to bring up connection %q: %q", i+1, tries, con.Connection.ID, string(output))
	}
	return fmt.Errorf("failed after %d attempts to bring up connection %q", tries, con.Connection.ID)
}

// convenience type for NetworkManager maps
type NMConnection struct {
	Connection NMInnerConnection
	active     bool
	raw        gonetworkmanager.Connection
}

// convenience type for NetworkManager maps
type NMInnerConnection struct {
	ID   string
	Type string
	UUID string
}
