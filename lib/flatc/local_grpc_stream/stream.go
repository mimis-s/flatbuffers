package local_grpc_stream

import (
	"context"
	"google.golang.org/grpc/metadata"
)

type queue struct {
	cap      int
	dataList chan any
}

func newQueue(cap int) *queue {
	q := &queue{
		cap:      cap,
		dataList: make(chan any, cap),
	}
	return q
}

func (q *queue) send(data any) error {
	q.dataList <- data
	return nil
}

func (q *queue) recv() (data any, err error) {
	data = <-q.dataList
	return data, nil
}

//type ServerStream interface {
//	SetHeader(metadata.MD) error
//	SendHeader(metadata.MD) error
//	SetTrailer(metadata.MD)
//	Context() context.Context
//	SendMsg(m any) error
//	RecvMsg(m any) error
//	Send(any) error
//	Recv() (any, error)
//}
//
//type ClientStream interface {
//	Header() (metadata.MD, error)
//	Trailer() metadata.MD
//	CloseSend() error
//	Context() context.Context
//	SendMsg(m any) error
//	RecvMsg(m any) error
//	Send(any) error
//	Recv() (any, error)
//}

func NewStream() (*LocalClientStream, *LocalServerStream) {
	c2sQueue := newQueue(128)
	s2cQueue := newQueue(128)

	cs := &LocalClientStream{
		c2sQueue: c2sQueue,
		s2cQueue: s2cQueue,
	}
	ss := &LocalServerStream{
		c2sQueue: c2sQueue,
		s2cQueue: s2cQueue,
	}
	return cs, ss
}

type LocalClientStream struct {
	*CommonClientStream
	c2sQueue *queue
	s2cQueue *queue
}

type CommonServerStream struct {
}

type CommonClientStream struct {
}

func (s *CommonClientStream) Header() (metadata.MD, error) {
	return nil, nil
}
func (s *CommonClientStream) Trailer() metadata.MD {
	return nil
}
func (s *CommonClientStream) CloseSend() error {
	return nil
}
func (s *CommonClientStream) Context() context.Context {
	ctx, _ := context.WithCancel(context.Background())
	return ctx
}
func (s *LocalClientStream) SendMsg(m any) error {
	return s.c2sQueue.send(m)
}
func (s *LocalClientStream) Send(data any) error {
	return s.c2sQueue.send(data)
}
func (s *LocalClientStream) Recv() (any, error) {
	return s.s2cQueue.recv()
}

type LocalServerStream struct {
	*CommonServerStream
	c2sQueue *queue
	s2cQueue *queue
}

func (s *CommonServerStream) SetHeader(metadata.MD) error {
	return nil
}
func (s *CommonServerStream) SendHeader(metadata.MD) error {
	return nil
}
func (s *CommonServerStream) SetTrailer(metadata.MD) {
	return
}
func (s *CommonServerStream) Context() context.Context {
	ctx, _ := context.WithCancel(context.Background())
	return ctx
}
func (s *LocalServerStream) SendMsg(m any) error {
	return s.s2cQueue.send(m)
}
func (s *LocalServerStream) Send(data any) error {
	return s.s2cQueue.send(data)
}
func (s *LocalServerStream) Recv() (any, error) {
	return s.c2sQueue.recv()
}
