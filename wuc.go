package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gobot.io/x/gobot/drivers/i2c"
)

const (
	cmdGetMoisture     = 0x10
	cmdGetWaterLevel   = 0x11
	cmdGetLastWatering = 0x12
	cmdGetWaterLimit   = 0x13
	cmdWatering        = 0x5A
)

// A Wuc provides the interface to the Watering Micro Controller.
type Wuc struct {
	connection i2c.Connection
	mutex      *sync.Mutex
}

// NewWuc creates an instance of a Wuc.
func NewWuc(c i2c.Connector) (*Wuc, error) {
	connection, err := c.GetConnection(0x10, 1)
	if err != nil {
		return nil, err
	}

	return &Wuc{
		connection: connection,
		mutex:      &sync.Mutex{},
	}, nil
}

// ReadMoisture triggers read of soil moisture.
func (w *Wuc) ReadMoisture() (m int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if err = w.connection.WriteByte(cmdGetMoisture); err != nil {
		return
	}

	time.Sleep(1000 * time.Millisecond)

	var buf [2]byte
	n, err := w.connection.Read(buf[:])
	if err != nil {
		return
	}

	if n != 2 {
		return 0, fmt.Errorf("invalid result length: %d", n)
	}

	if buf[1] == 0xFF {
		return 0, fmt.Errorf("failed to measure soil moisture")
	}

	m = (int(buf[1]) << 8) | int(buf[0])

	return
}

// DoWatering sends command for watering.
func (w *Wuc) DoWatering(ms int) int {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	u := (ms + 125) / 250
	if u < 0 || u > 255 {
		log.Printf("watering time out of range: %v(%v)", u, ms)
		return 0
	}

	log.Printf("watering %v ms", u*250)
	cmd := []byte{cmdWatering, byte(u)}

	n, err := w.connection.Write(cmd)
	if err != nil {
		log.Printf("failed to send watering command: %v", err)
		return 0
	}

	if n < len(cmd) {
		log.Printf("could not send complete watering command: %v/%v", n, len(cmd))
		return 0
	}

	// wait for watering to finish and some margin
	time.Sleep(time.Duration(ms+500) * time.Millisecond)

	r, err := w.connection.ReadByte()

	if err != nil {
		log.Printf("failed to read watering time: %v", err)
		return 0
	}

	if int(r) != u {
		log.Printf("watered %v ms", int(r)*250)
	}

	return int(r) * 250
}

// ReadLastWatering queries duration of last watering and returns time in ms.
func (w *Wuc) ReadLastWatering() (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if err := w.connection.WriteByte(cmdGetLastWatering); err != nil {
		return 0, err
	}

	t, err := w.connection.ReadByte()
	if err != nil {
		return 0, err
	}

	if t == 0xFF {
		return 0, fmt.Errorf("failed to get last watering time")
	}

	return int(t) * 250, nil
}

// ReadWaterLevel sends command to measure water level and returns result.
func (w *Wuc) ReadWaterLevel() (l int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if err = w.connection.WriteByte(cmdGetWaterLevel); err != nil {
		return
	}

	time.Sleep(1000 * time.Millisecond)

	var buf [2]byte
	n, err := w.connection.Read(buf[:])
	if err != nil {
		return
	}

	if n != 2 {
		return 0, fmt.Errorf("invalid result length: %d", n)
	}

	if buf[1] == 0xFF {
		return 0, fmt.Errorf("failed to measure water level")
	}

	l = (int(buf[1]) << 8) | int(buf[0])

	return
}
