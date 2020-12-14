package watcher

import (
	"bytes"
	"errors"
)

type ReadRegister func(slaveID byte, address uint16, quantity uint16) (results []byte, err error)

type Config struct {
	Address      uint16
	Quantity     uint16
	SlaveID      byte
	Read         ReadRegister
	RegisterSize int
}

type Watcher struct {
	Config
	state     []byte
	callbacks map[uint16]func(address uint16)
}

var ErrIncorrectRegisterSize = errors.New("Incorrect register size")
var ErrAddressOutOfRange = errors.New("Register address out of range")
var ErrUninitialized = errors.New("State uninitialized. Call Poll() first.")

func New(config *Config) *Watcher {
	return &Watcher{
		Config:    *config,
		callbacks: make(map[uint16]func(address uint16)),
	}
}

func (w *Watcher) RegisterCallback(address uint16, callback func(address uint16)) {
	w.callbacks[address] = callback
}

func (w *Watcher) Poll() error {
	newState, err := w.Read(w.SlaveID, w.Address, w.Quantity)
	if err != nil {
		return err
	}

	if len(newState) != int(w.Quantity)*w.RegisterSize {
		return ErrIncorrectRegisterSize
	}

	oldState := w.state
	w.state = newState

	first := len(oldState) != len(newState)
	address := w.Address
	for n := 0; n < len(newState); n += w.RegisterSize {
		callback := w.callbacks[address]
		if callback == nil {
			address++
			continue
		}
		var oldValue []byte
		newValue := newState[n : n+w.RegisterSize]
		if first {
			oldValue = nil
		} else {
			oldValue = oldState[n : n+w.RegisterSize]
		}
		if bytes.Compare(oldValue, newValue) != 0 {
			callback(address)
		}
		address++
	}
	return nil
}

func (w *Watcher) ReadRegister(address uint16) (value []byte) {
	if address < w.Address || address > w.Address+uint16(w.Quantity) {
		panic(ErrAddressOutOfRange)
	}
	if w.state == nil {
		panic(ErrUninitialized)
	}
	registerOffset := int(address-w.Address) * w.RegisterSize
	return w.state[registerOffset : registerOffset+w.RegisterSize]

}

func (w *Watcher) TriggerCallbacks() {
	for address, callback := range w.callbacks {
		callback(address)
	}
}
