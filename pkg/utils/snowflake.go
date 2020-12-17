package utils

import (
	"errors"
	"sync"
	"time"

	"gosm/pkg/log"
)

// 1                                               42           52             64
// +-----------------------------------------------+------------+---------------+
// | timestamp(ms)                                 | workerid   | sequence      |
// +-----------------------------------------------+------------+---------------+
// | 0000000000 0000000000 0000000000 0000000000 0 | 0000000000 | 0000000000 00 |
// +-----------------------------------------------+------------+---------------+

// 1. 41位时间截(毫秒级)，注意这是时间截的差值（当前时间截 - 开始时间截)。可以使用约70年: (1L << 41) / (1000L * 60 * 60 * 24 * 365) = 69
// 2. 10位数据机器位，可以部署在1024个节点
// 3. 12位序列，毫秒内的计数，同一机器，同一时间截并发4096个序号
const (
	// 工作ID长度
	workerIDBits int64 = 5
	// 数据中心ID长度
	datacenterIDBits int64 = 5
	// 序列号长度
	sequenceBits int64 = 12

	// 最大工作ID(31)
	maxWorkerID int64 = -1 ^ (-1 << uint64(workerIDBits))
	/** 最大数据中心ID(31) */
	maxDatacenterID int64 = -1 ^ (-1 << uint64(datacenterIDBits))
	/** 最大序列号(4095) */
	maxSequence int64 = -1 ^ (-1 << uint64(sequenceBits))

	// 工作ID需要左移的位数:12位
	workShift uint8 = 12
	// 数据中心ID需要左移位数:12+5=17位
	dataShift uint8 = 17
	// 时间戳需要左移位数:12+5+5=22位
	timeShift uint8 = 22

	/** 初始时间戳 2020-01-01 */
	startTimestamp int64 = 1577808000000
)

// IDWorker id worker
type IDWorker struct {
	mu            sync.Mutex
	lastTimestamp int64
	workerID      int64
	datacenterID  int64
	sequence      int64
}

// NewSnowflake returns a new snowflake worker that can be used to generate snowflake IDs
func NewSnowflake(workerID int64, datacenterID int64) (*IDWorker, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, errors.New("workerID must be between 0 and 31")
	}
	if datacenterID < 0 || datacenterID > maxDatacenterID {
		return nil, errors.New("datacenterID must be between 0 and 31")
	}

	idWorker := &IDWorker{
		lastTimestamp: -1,
		workerID:      workerID,
		datacenterID:  datacenterID,
		sequence:      0,
	}
	return idWorker, nil
}

// NextID get next woker id
func (w *IDWorker) NextID() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 获取当前时间戳如果小于上次时间戳,则表示时间戳获取出现异常
	timestamp := w.getCurrentTime()
	if timestamp < w.lastTimestamp {
		log.Fatal("clock is moving backwards, rejecting requests until %d", w.lastTimestamp)
	}

	// 获取当前时间戳如果等于上次时间戳(同一毫秒内),则在序列号加一;否则序列号赋值为0,从0开始。
	if w.lastTimestamp == timestamp {
		w.sequence = (w.sequence + 1) & maxSequence
		if w.sequence == 0 {
			timestamp = w.tilNextMillis()
		}
	} else {
		w.sequence = 0
	}
	// 将上次时间戳值刷新
	w.lastTimestamp = timestamp

	// 时间戳部分 + 机器标识部分 + 序列号部分
	return ((timestamp - startTimestamp) << timeShift) |
		(w.datacenterID << dataShift) |
		(w.workerID << workShift) |
		w.sequence
}

// 返回以毫秒为单位的当前时间
func (w *IDWorker) getCurrentTime() int64 {
	return time.Now().UnixNano() / 1e6
}

// 阻塞到下一个毫秒,获得新的时间戳
func (w *IDWorker) tilNextMillis() int64 {
	timestamp := w.getCurrentTime()
	for timestamp <= w.lastTimestamp {
		timestamp = w.getCurrentTime()
	}
	return timestamp
}
