package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gobot.io/x/gobot/drivers/i2c"
)

const (
	cmdGetLastWatering = 0x10
	cmdGetWaterLimit   = 0x11
	cmdGetWeight       = 0x12
	cmdWatering        = 0x1A
	cmdEcho            = 0x29
)

const cmdShift = 1
const cmdMask = 0xFF << cmdShift

func consCmd(cmd byte, index int) byte {
	return (cmd << cmdShift) | (byte)(index & ^cmdMask)
}

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

// ReadWeights triggers read of weight sensors.
func (w *Wuc) ReadWeights() (m1 int, m2 int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if err = w.connection.WriteByte(consCmd(cmdGetWeight, 0)); err != nil {
		return
	}

	time.Sleep(700 * time.Millisecond)

	var buf [2]byte
	n, err := w.connection.Read(buf[:])
	if err != nil {
		return
	}

	if n != 2 {
		return 0, 0, fmt.Errorf("invalid length of result #1: %d", n)
	}

	if buf[1] == 0xFF {
		return 0, 0, fmt.Errorf("failed to measure weight #1")
	}

	m1 = (int(buf[1]) << 8) | int(buf[0])

	if err = w.connection.WriteByte(consCmd(cmdGetWeight, 1)); err != nil {
		return
	}

	time.Sleep(700 * time.Millisecond)

	n, err = w.connection.Read(buf[:])
	if err != nil {
		return
	}

	if n != 2 {
		return 0, 0, fmt.Errorf("invalid length of result #2: %d", n)
	}

	if buf[1] == 0xFF {
		return 0, 0, fmt.Errorf("failed to measure weight #2")
	}

	m2 = (int(buf[1]) << 8) | int(buf[0])

	return
}

// DoWatering sends command for watering.
func (w *Wuc) DoWatering(index, ms int) int {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	u := (ms + 125) / 250
	if u < 0 || u > 255 {
		log.Printf("watering time out of range: %v(%v)", u, ms)
		return 0
	}

	log.Printf("watering %v ms", u*250)
	cmd := []byte{consCmd(cmdWatering, index), byte(u)}

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
func (w *Wuc) ReadLastWatering(index int) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if err := w.connection.WriteByte(consCmd(cmdGetLastWatering, index)); err != nil {
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

// ReadWateringLimit sends command to measure water Limit and returns result.
func (w *Wuc) ReadWateringLimit(index int) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if err := w.connection.WriteByte(consCmd(cmdGetWaterLimit, index)); err != nil {
		return 0, err
	}

	l, err := w.connection.ReadByte()
	if err != nil {
		return 0, err
	}

	if l == 0xFF {
		return 0, fmt.Errorf("failed to measure water Limit")
	}

	return int(l), nil
}

// Echo sends echo command with data of given buffer and returns result.
func (w *Wuc) Echo(buf []byte) ([]byte, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	b := make([]byte, len(buf)+1)
	b[0] = consCmd(cmdEcho, 0)
	copy(b[1:], buf)

	if _, err := w.connection.Write(b); err != nil {
		return nil, err
	}

	n, err := w.connection.Read(b)
	if err != nil {
		return nil, err
	}

	b = b[:n]

	return b, err
}
