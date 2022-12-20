package main

import (
	"context"
	"fmt"
	"github.com/jrwalls/chat-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"log"
	"net"
	"sync"
)

type (
	user struct {
		username string
		addr     string
		chTell   chan *proto.Tell
	}

	server struct {
		users     map[string]*user //k = addr, v = user
		userMutex sync.RWMutex
		proto.UnimplementedChatServiceServer
	}
)

func (s *server) Login(ctx context.Context, in *proto.LoginRequest) (*proto.LoginResponse, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no peer info in context")
	}

	addr := p.Addr.String()
	log.Printf("user %s connected from %s", in.Username, addr)

	// todo check name not in user map
	s.userMutex.Lock()
	s.users[addr] = &user{
		username: in.Username,
		addr:     addr,
		chTell:   make(chan *proto.Tell, 8), // 8 pending tells
	}
	s.userMutex.Unlock()

	return &proto.LoginResponse{Status: ""}, nil
}

func (s *server) SubscribeToChannel(in proto.ChatService_SubscribeToChannelServer) error {
	p, ok := peer.FromContext(in.Context())
	if !ok {
		return fmt.Errorf("no peer info in context")
	}

	s.userMutex.RLock()
	user, ok := s.users[p.Addr.String()]
	s.userMutex.RUnlock()
	if !ok {
		return fmt.Errorf("no user connected")
	}

	go func() {
		for t := range user.chTell {
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
			c.Message.Sender = user.username
			err := s.broadcastMessage(c.Message)
			if err != nil {
				return err
			}
		}
	}
}

func (s *server) broadcastMessage(m *proto.Message) error {
	log.Printf("%s: %s", m.Sender, m.Body)
	s.userMutex.RLock()
	defer s.userMutex.RUnlock()
	for _, user := range s.users {
		user.chTell <- &proto.Tell{
			Id: "hi",
			Tell: &proto.Tell_Message{
				Message: m,
			},
		}
	}
	return nil
}

func main() {
	grpcServer := grpc.NewServer()
	s := &server{
		users: map[string]*user{},
	}
	proto.RegisterChatServiceServer(grpcServer, s)

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	//go func() {
	//	for range time.Tick(5 * time.Second) {
	//		err := s.broadcastMessage(&proto.Message{
	//			Body:   "fake message",
	//			Sender: "server",
	//		})
	//		if err != nil {
	//			log.Print("unable to send message: ", err)
	//		}
	//	}
	//}()

	if err = grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
