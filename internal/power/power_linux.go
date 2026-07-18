//go:build linux

package power

import "os/exec"

func acquire() (func(), bool, error) {
	path, err := exec.LookPath("systemd-inhibit")
	if err != nil {
		return func() {}, false, nil
	}
	cmd := exec.Command(path, "--what=idle:sleep", "--mode=block", "--why=VCSM broker is unlocked", "sleep", "infinity")
	if err := cmd.Start(); err != nil {
		return nil, false, err
	}
	return func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}, true, nil
}
