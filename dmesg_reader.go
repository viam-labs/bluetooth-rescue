package bluetoothrescue

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
)

// lightly parsed dmesg line
type DmesgMesg struct {
	Timestamp string
	Message   string
}

var regexpDmesg = regexp.MustCompile(`^\[([\s\d\.]+)\] (.+)$`)

func DmesgReader(ctx context.Context, ch chan DmesgMesg) error {
	cmd := exec.CommandContext(ctx, "dmesg", "--follow")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		matches := regexpDmesg.FindStringSubmatch(scanner.Text())
		if matches == nil {
			// todo: warn here, but exclude some normal ones first
			continue
		}
		ch <- DmesgMesg{matches[1], matches[2]}
	}

	return scanner.Err()
}
