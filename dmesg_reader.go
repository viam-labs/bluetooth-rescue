package bluetoothrescue

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
)

// lightly parsed dmesg line
type DmesgLine struct {
	Timestamp string
	Message   string
}

// parses a dmesg log line like `[123.456] msg msg msg`
var regexpDmesg = regexp.MustCompile(`^\[([\s\d\.]+)\] (.+)$`)

// watches `dmesg --follow` and writes DmesgLine structs to `ch`
func DmesgReader(ctx context.Context, ch chan DmesgLine) error {
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
			// todo: warn here, but exclude some normal unparseable lines first
			continue
		}
		ch <- DmesgLine{matches[1], matches[2]}
	}

	return scanner.Err()
}
