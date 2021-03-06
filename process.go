package goworker

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type process struct {
	Hostname string
	Pid      int
	Id       string
	Queues   []string
}

func newProcess(id string, queues []string) (*process, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &process{
		Hostname: hostname,
		Pid:      os.Getpid(),
		Id:       id,
		Queues:   queues,
	}, nil
}

func (p *process) String() string {
	return fmt.Sprintf("%s:%d-%s:%s", p.Hostname, p.Pid, p.Id, strings.Join(p.Queues, ","))
}

func (p *process) open(conn *redisConn) error {
	conn.Send("SADD", fmt.Sprintf("%sworkers", cfg.namespace), p)
	conn.Send("SET", fmt.Sprintf("%sstat:processed:%v", cfg.namespace, p), "0")
	conn.Send("SET", fmt.Sprintf("%sstat:failed:%v", cfg.namespace, p), "0")
	conn.Flush()

	return nil
}

func (p *process) close(conn *redisConn) error {
	logger.Infof("%v shutdown", p)
	conn.Send("SREM", fmt.Sprintf("%sworkers", cfg.namespace), p)
	conn.Send("DEL", fmt.Sprintf("%sstat:processed:%s", cfg.namespace, p))
	conn.Send("DEL", fmt.Sprintf("%sstat:failed:%s", cfg.namespace, p))
	conn.Flush()

	return nil
}

func (p *process) start(conn *redisConn) error {
	conn.Send("SET", fmt.Sprintf("%sworker:%s:started", cfg.namespace, p), time.Now().String())
	conn.Flush()

	return nil
}

func (p *process) finish(conn *redisConn) error {
	conn.Send("DEL", fmt.Sprintf("%sworker:%s", cfg.namespace, p))
	conn.Send("DEL", fmt.Sprintf("%sworker:%s:started", cfg.namespace, p))
	conn.Flush()

	return nil
}

func (p *process) fail(conn *redisConn) error {
	conn.Send("INCR", fmt.Sprintf("%sstat:failed", cfg.namespace))
	conn.Send("INCR", fmt.Sprintf("%sstat:failed:%s", cfg.namespace, p))
	conn.Flush()

	return nil
}

func (p *process) queues(strict bool) []string {
	// If the queues order is strict then just return them.
	if strict {
		return p.Queues
	}

	// If not then we want to to shuffle the queues before returning them.
	queues := make([]string, len(p.Queues))
	for i, v := range rand.Perm(len(p.Queues)) {
		queues[i] = p.Queues[v]
	}
	return queues
}
