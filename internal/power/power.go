package power

import "sync"

type Manager struct {
	mu     sync.Mutex
	active bool
	stop   func()
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active {
		return nil
	}
	stop, active, err := acquire()
	if err != nil {
		return err
	}
	m.stop = stop
	m.active = active
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stop != nil {
		m.stop()
	}
	m.stop = nil
	m.active = false
}

func (m *Manager) Active() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active
}
