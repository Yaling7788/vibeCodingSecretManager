//go:build darwin

package power

import "os/exec"

func acquire() (func(), bool, error) {
	// -i prevents idle system sleep but intentionally permits display sleep.
	cmd := exec.Command("caffeinate", "-i")
	if err := cmd.Start(); err != nil {
		return nil, false, err
	}
	return func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}, true, nil
}
