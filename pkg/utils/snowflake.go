package utils

import (
	"errors"
	"sync"
	"time"

	"gosm/pkg/config"
	"gosm/pkg/log"
)

// Snowflake golbal ID generator
var Snowflake, _ = NewSnowflake(config.Global.MachineID, config.Global.DataCenterID)

// 1                                               42           52             64
// +-----------------------------------------------+-------------+---------------+
// | timestamp(ms)                                 |  workerid   | sequence      |
// +-----------------------------------------------+-------------+---------------+
// | 0000000000 0000000000 0000000000 0000000000 0 | 00000 00000 | 0000000000 00 |
// +-----------------------------------------------+-------------+---------------+

const (
	MachineIDBits    int64 = 5
	DatacenterIDBits int64 = 5
	SequenceBits     int64 = 12

	MaxMachineID    int64 = -1 ^ (-1 << uint64(MachineIDBits))
	MaxDatacenterID int64 = -1 ^ (-1 << uint64(DatacenterIDBits))
	MaxSequence     int64 = -1 ^ (-1 << uint64(SequenceBits))

	TimeShift    uint8 = 22
	DataShift    uint8 = 17
	MachineShift uint8 = 12

	StartTimestamp int64 = 1577808000000 // 2020-01-01
)

// IDWorker id worker
type IDWorker struct {
	mu            sync.Mutex
	lastTimestamp int64
	machineID     int64
	datacenterID  int64
	sequence      int64
}

// NewSnowflake returns worker to generate snowflake IDs
func NewSnowflake(machineID int64, datacenterID int64) (*IDWorker, error) {
	if machineID < 0 || machineID > MaxMachineID {
		return nil, errors.New("workerID must be between 0 and 31")
	}
	if datacenterID < 0 || datacenterID > MaxDatacenterID {
		return nil, errors.New("datacenterID must be between 0 and 31")
	}

	idWorker := &IDWorker{
		lastTimestamp: -1,
		machineID:     machineID,
		datacenterID:  datacenterID,
		sequence:      0,
	}
	return idWorker, nil
}

// NextID get next woker id
func (w *IDWorker) NextID() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	// timestamp should not go back
	timestamp := w.getCurrentTime()
	if timestamp < w.lastTimestamp {
		log.Fatal("clock is moving backwards, rejecting requests until %d", w.lastTimestamp)
	}

	// increases sequence number whthin one ms
	if w.lastTimestamp == timestamp {
		w.sequence = (w.sequence + 1) & MaxSequence
		if w.sequence == 0 {
			timestamp = w.tilNextMillis()
		}
	} else {
		w.sequence = 0
	}
	w.lastTimestamp = timestamp

	// timestamp + worker + sequence
	return ((timestamp - StartTimestamp) << TimeShift) |
		(w.datacenterID << DataShift) |
		(w.machineID << MachineShift) |
		w.sequence
}

func (w *IDWorker) getCurrentTime() int64 {
	return time.Now().UnixNano() / 1e6
}

func (w *IDWorker) tilNextMillis() int64 {
	timestamp := w.getCurrentTime()
	for timestamp <= w.lastTimestamp {
		timestamp = w.getCurrentTime()
	}
	return timestamp
}
