package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"
)

type MyMicroservice struct {
	mu        sync.RWMutex
	stat      map[int]Stat
	event     []chan Event
	acl       map[string][]string
	clientNum int
}

type Evt struct {
	Method   string
	Consumer string
	Host     string
}

func NewMyMicroservice() *MyMicroservice {
	return &MyMicroservice{mu: sync.RWMutex{}, stat: map[int]Stat{}}
}

func parseACLData(acl string) (map[string][]string, error) {
	var d map[string][]string
	err := json.Unmarshal([]byte(acl), &d)

	return d, err
}

func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	acl, err := parseACLData(ACLData)
	if err != nil {
		return fmt.Errorf("Bad ACLData")
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	mm := NewMyMicroservice()
	mm.acl = acl

	RegisterAdminServer(server, mm)
	RegisterBizServer(server, mm)

	go server.Serve(lis)

	go func() {
		select {
		case <-ctx.Done():
			server.Stop()
			for _, e := range mm.event {
				close(e)
			}
			return
		}
	}()

	return nil
}

func GetBizParams(ctx context.Context) (e error, m string, c string, h string) {
	md, _ := metadata.FromIncomingContext(ctx)
	consumer := md.Get("consumer")
	host := md.Get(":authority")
	m, _ = grpc.Method(ctx)
	if consumer == nil {
		e = grpc.Errorf(codes.Unauthenticated, "unknown consumer")
		return
	}
	if host == nil {
		e = grpc.Errorf(codes.Unauthenticated, "unknown host")
		return
	}

	h = host[0]
	c = consumer[0]

	return
}

func GetAdminParams(inStream grpc.ServerStream) (e error, m string, c string, h string) {
	m, _ = grpc.MethodFromServerStream(inStream)
	md, _ := metadata.FromIncomingContext(inStream.Context())
	consumer := md.Get("consumer")
	host := md.Get(":authority")

	if consumer == nil {
		e = grpc.Errorf(codes.Unauthenticated, "unknown consumer")
		return
	}
	if host == nil {
		e = grpc.Errorf(codes.Unauthenticated, "unknown host")
		return
	}

	h = host[0]
	c = consumer[0]

	return
}

func (srv *MyMicroservice) Stat(c string, m string) error {
	srv.mu.Lock()
	for _, st := range srv.stat {
		st.Timestamp = time.Now().Unix()
		if _, ok := st.ByMethod[m]; ok {
			st.ByMethod[m] += 1
		} else {
			st.ByMethod[m] = 1
		}
		if _, ok := st.ByConsumer[c]; ok {
			st.ByConsumer[c] += 1
		} else {
			st.ByConsumer[c] = 1
		}
	}
	srv.mu.Unlock()

	return nil
}

func (srv *MyMicroservice) Log(c string, m string, h string) error {
	srv.mu.Lock()

	for _, e := range srv.event {
		e <- Event{Consumer: c, Method: m, Host: h[0 : strings.LastIndex(h, ":")+1]}
	}
	srv.mu.Unlock()

	return nil
}

func (srv *MyMicroservice) ACLCheck(c string, m string) error {
	if methods, ok := srv.acl[c]; ok {
		for _, mthd := range methods {
			if strings.HasSuffix(mthd, "*") && strings.HasPrefix(m, mthd[:len(mthd)-1]) || mthd == m {
				return nil
			}
		}
	}
	return grpc.Errorf(codes.Unauthenticated, "unknown consumer")
}

func (srv *MyMicroservice) Check(ctx context.Context, in *Nothing) (*Nothing, error) {
	err, m, c, h := GetBizParams(ctx)
	if err != nil {
		return nil, err
	}

	srv.Stat(c, m)
	srv.Log(c, m, h)

	err = srv.ACLCheck(c, m)
	if err != nil {
		return nil, err
	}

	return &Nothing{}, nil
}

func (srv *MyMicroservice) Add(ctx context.Context, in *Nothing) (*Nothing, error) {
	err, m, c, h := GetBizParams(ctx)
	if err != nil {
		return nil, err
	}

	srv.Stat(c, m)
	srv.Log(c, m, h)

	err = srv.ACLCheck(c, m)
	if err != nil {
		return nil, err
	}

	return &Nothing{}, nil
}

func (srv *MyMicroservice) Test(ctx context.Context, in *Nothing) (*Nothing, error) {
	err, m, c, h := GetBizParams(ctx)
	if err != nil {
		return nil, err
	}

	srv.Stat(c, m)
	srv.Log(c, m, h)

	err = srv.ACLCheck(c, m)
	if err != nil {
		return nil, err
	}

	return &Nothing{}, nil
}

func (srv *MyMicroservice) Logging(in *Nothing, inStream Admin_LoggingServer) error {
	err, m, c, h := GetAdminParams(inStream)
	if err != nil {
		return err
	}

	srv.Stat(c, m)
	srv.Log(c, m, h)

	err = srv.ACLCheck(c, m)
	if err != nil {
		return err
	}

	evtCh := make(chan Event, 1)
	srv.mu.Lock()
	srv.event = append(srv.event, evtCh)
	srv.mu.Unlock()

	for {
		e, isOpen := <-evtCh
		if !isOpen {
			break
		}
		inStream.Send(&e)
	}
	return nil
}

func (srv *MyMicroservice) Statistics(in *StatInterval, inStream Admin_StatisticsServer) error {
	err, m, c, h := GetAdminParams(inStream)
	if err != nil {
		return err
	}

	srv.Stat(c, m)
	srv.Log(c, m, h)

	err = srv.ACLCheck(c, m)
	if err != nil {
		return err
	}

	i := srv.clientNum

	srv.mu.Lock()
	srv.stat[srv.clientNum] = Stat{ByMethod: map[string]uint64{}, ByConsumer: map[string]uint64{}}
	srv.clientNum += 1
	srv.mu.Unlock()

	ticker := time.NewTicker(time.Duration(in.IntervalSeconds) * time.Second)
	for _ = range ticker.C {
		srv.mu.Lock()
		out := srv.stat[i]
		srv.stat[i] = Stat{ByMethod: map[string]uint64{}, ByConsumer: map[string]uint64{}}
		inStream.Send(&out)
		srv.mu.Unlock()
	}

	return nil
}
