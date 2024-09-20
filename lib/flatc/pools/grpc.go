package pools

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var (
	DefaultGRPCPoolSize       = 1000
	DefaultGRPCPoolTTL        = time.Minute
	DefaultGRPCPoolMaxStreams = 10
	DefaultGRPCPoolMaxIdle    = 200
)

type GRPCPoolOption struct {
	PoolSize      int
	PoolTTL       time.Duration
	PoolMaxStream int
	PoolMaxIdle   int
}

var DefaultGRPCPoolOption = GRPCPoolOption{
	PoolSize:      DefaultGRPCPoolSize,
	PoolTTL:       DefaultGRPCPoolTTL,
	PoolMaxStream: DefaultGRPCPoolMaxStreams,
	PoolMaxIdle:   DefaultGRPCPoolMaxIdle,
}

type GRPCPool struct {
	size int
	ttl  int64

	//  max streams on a *grpcPoolConn
	maxStreams int
	//  max idle conns
	maxIdle int

	sync.Mutex
	conns map[string]*grpcStreamsPool
}

type grpcStreamsPool struct {
	//  head of list
	head *grpcPoolConn
	//  busy conns list
	busy *grpcPoolConn
	//  the siza of list
	count int
	//  idle conn
	idle int
}

type grpcPoolConn struct {
	//  grpc conn
	*grpc.ClientConn
	err  error
	addr string

	//  pool and streams pool
	pool    *GRPCPool
	sp      *grpcStreamsPool
	streams int
	created int64

	//  list
	pre  *grpcPoolConn
	next *grpcPoolConn
	in   bool
}

func NewGRPCPool(option GRPCPoolOption) *GRPCPool {
	if option.PoolMaxStream <= 0 {
		option.PoolMaxStream = 1
	}
	if option.PoolMaxIdle < 0 {
		option.PoolMaxIdle = 0
	}
	return &GRPCPool{
		size:       option.PoolSize,
		ttl:        int64(option.PoolTTL.Seconds()),
		maxStreams: option.PoolMaxStream,
		maxIdle:    option.PoolMaxIdle,
		conns:      make(map[string]*grpcStreamsPool),
	}
}

func (p *GRPCPool) GetConn(dialCtx context.Context, addr string, opts ...grpc.DialOption) (*grpcPoolConn, error) {
	now := time.Now().Unix()
	p.Lock()
	sp, ok := p.conns[addr]
	if !ok {
		sp = &grpcStreamsPool{head: &grpcPoolConn{}, busy: &grpcPoolConn{}, count: 0, idle: 0}
		p.conns[addr] = sp
	}
	//  while we have conns check streams and then return one
	//  otherwise we'll create a new conn
	conn := sp.head.next
	for conn != nil {
		//  check conn state
		// https://github.com/grpc/grpc/blob/master/doc/connectivity-semantics-and-api.md
		switch conn.GetState() {
		case connectivity.Connecting:
			conn = conn.next
			continue
		case connectivity.Shutdown:
			next := conn.next
			if conn.streams == 0 {
				removeConn(conn)
				sp.idle--
			}
			conn = next
			continue
		case connectivity.TransientFailure:
			next := conn.next
			if conn.streams == 0 {
				removeConn(conn)
				conn.ClientConn.Close()
				sp.idle--
			}
			conn = next
			continue
		case connectivity.Ready:
		case connectivity.Idle:
		}
		//  a old conn
		if now-conn.created > p.ttl {
			next := conn.next
			if conn.streams == 0 {
				removeConn(conn)
				conn.ClientConn.Close()
				sp.idle--
			}
			conn = next
			continue
		}
		//  a busy conn
		if conn.streams >= p.maxStreams {
			next := conn.next
			removeConn(conn)
			addConnAfter(conn, sp.busy)
			conn = next
			continue
		}
		//  a idle conn
		if conn.streams == 0 {
			sp.idle--
		}
		//  a good conn
		conn.streams++
		p.Unlock()
		return conn, nil
	}
	p.Unlock()
	//  create new conn
	cc, err := grpc.DialContext(dialCtx, addr, opts...)
	if err != nil {
		return nil, err
	}
	conn = &grpcPoolConn{cc, nil, addr, p, sp, 1, time.Now().Unix(), nil, nil, false}

	//  add conn to streams GRPCPool
	p.Lock()
	if sp.count < p.size {
		addConnAfter(conn, sp.head)
	}
	p.Unlock()

	return conn, nil
}

func (p *GRPCPool) release(conn *grpcPoolConn, err error) {
	p.Lock()
	p, sp, created := conn.pool, conn.sp, conn.created
	//  try to add conn
	if !conn.in && sp.count < p.size {
		addConnAfter(conn, sp.head)
	}
	if !conn.in {
		p.Unlock()
		conn.ClientConn.Close()
		return
	}
	//  a busy conn
	if conn.streams >= p.maxStreams {
		removeConn(conn)
		addConnAfter(conn, sp.head)
	}
	conn.streams--
	//  if streams == 0, we can do something
	if conn.streams == 0 {
		//  1. it has errored
		//  2. too many idle conn or
		//  3. conn is too old
		now := time.Now().Unix()
		if err != nil || sp.idle >= p.maxIdle || now-created > p.ttl {
			removeConn(conn)
			p.Unlock()
			conn.ClientConn.Close()
			return
		}
		sp.idle++
	}
	p.Unlock()
}

func (conn *grpcPoolConn) Close() {
	conn.pool.release(conn, conn.err)
}

func removeConn(conn *grpcPoolConn) {
	if conn.pre != nil {
		conn.pre.next = conn.next
	}
	if conn.next != nil {
		conn.next.pre = conn.pre
	}
	conn.pre = nil
	conn.next = nil
	conn.in = false
	conn.sp.count--
	return
}

func addConnAfter(conn *grpcPoolConn, after *grpcPoolConn) {
	conn.next = after.next
	conn.pre = after
	if after.next != nil {
		after.next.pre = conn
	}
	after.next = conn
	conn.in = true
	conn.sp.count++
	return
}
