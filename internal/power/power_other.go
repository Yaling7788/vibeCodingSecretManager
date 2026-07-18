//go:build !darwin && !linux && !windows

package power

func acquire() (func(), bool, error) { return func() {}, false, nil }
