package main

import (
	"context"
	"fmt"
	"github.com/jrwalls/chat-server/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"log"
	"net"
	"sync"
)

type (
	address string
	status  string

	user struct {
		username string
		address  address
		chTell   chan *proto.Tell
	}

	server struct {
		users    map[address]*user
		userLock sync.RWMutex
		proto.UnimplementedChatServiceServer
	}
)

const (
	statusSuccess status = "SUCCESS"
	statusFailed  status = "FAILED"
)

func main() {
	grpcServer := grpc.NewServer()
	s := &server{
		users: map[address]*user{},
	}
	proto.RegisterChatServiceServer(grpcServer, s)

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	if err = grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}

func (s *server) Login(ctx context.Context, in *proto.LoginRequest) (*proto.LoginResponse, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no peer info in context")
	}

	addr := address(p.Addr.String())
	log.Printf("user %s connected from %s", in.Username, addr)

	s.userLock.Lock()
	s.users[addr] = &user{
		username: in.Username,
		address:  addr,
		chTell:   make(chan *proto.Tell, 8),
	}
	s.userLock.Unlock()

	return &proto.LoginResponse{Status: string(statusSuccess)}, nil
}

func (s *server) SubscribeToChannel(in proto.ChatService_SubscribeToChannelServer) error {
	p, ok := peer.FromContext(in.Context())
	if !ok {
		return fmt.Errorf("no peer info in context")
	}

	s.userLock.RLock()
	usr, ok := s.users[address(p.Addr.String())]
	s.userLock.RUnlock()
	if !ok {
		return fmt.Errorf("no usr connected")
	}

	go func() {
		for t := range usr.chTell {
			err := in.Send(t)
			if err != nil {
				log.Print("unable to send message: ", err)
			}
		}
	}()

	for {
		recv, err := in.Recv()
		if err != nil {
			return fmt.Errorf("can't recv: %w", err)
		}
		switch c := recv.Ask.(type) {
		case *proto.Command_Message:
			c.Message.Sender = usr.username
			err := s.Broadcast(c.Message)
			if err != nil {
				return err
			}
		}
	}
}

func (s *server) Broadcast(m *proto.Message) error {
	log.Printf("%s: %s", m.Sender, m.Body)
	s.userLock.RLock()
	defer s.userLock.RUnlock()
	for _, u := range s.users {
		u.chTell <- &proto.Tell{
			Id: "hi",
			Tell: &proto.Tell_Message{
				Message: m,
			},
		}
	}

	return nil
}

//go func() {
//	for range time.Tick(5 * time.Second) {
//		err := s.Broadcast(&proto.Message{
//			Body:   "fake message",
//			Sender: "server",
//		})
//		if err != nil {
//			log.Print("unable to send message: ", err)
//		}
//	}
//}()
