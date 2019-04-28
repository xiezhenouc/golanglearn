package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	seqBits    uint8 = 12
	workerBits uint8 = 10
	timeBits   uint8 = 41

	seqMax    int64 = -1 ^ (-1 << seqBits)
	workerMax int64 = -1 ^ (-1 << workerBits)

	timeShift uint8 = seqBits + workerBits
	workShift uint8 = seqBits

	// 发号器开始时间戳 2019-01-01 00:00:00
	epoch = 1546272000000
)

type Worker struct {
	mu      sync.Mutex
	curtime int64
	workid  int64
	seqid   int64
}

func NewWorker(workid int64) (*Worker, error) {
	if workid < 0 || workid > workerMax {
		return nil, errors.New("workid is not in [0, workMax]")
	}

	return &Worker{
		curtime: 0,
		workid:  workid,
		seqid:   0,
	}, nil
}

func (w *Worker) GetID() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 纳秒->毫秒
	now := time.Now().UnixNano() / 1e6
	if w.curtime == now {
		w.seqid++

		if w.seqid > seqMax {
			for now <= w.curtime {
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		w.curtime = now
		w.seqid = 0
	}

	ID := (int64(now-epoch) << timeShift) | (w.workid << workShift) | (w.seqid)
	return ID
}
