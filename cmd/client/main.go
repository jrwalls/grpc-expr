package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jrwalls/chat-server/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

type (
	User struct {
		config *config
		client proto.ChatServiceClient
		sub    proto.ChatService_SubscribeToChannelClient
	}
)

func main() {
	cfg := setConfig()
	flag.Parse()

	user, err := newUser(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err = user.Login(); err != nil {
		log.Fatal(err)
	}

	if err = user.Subscribe(); err != nil {
		log.Fatal(err)

	}

	user.SendRoutine()
	user.ReceiveRoutine()
}

func newUser(config *config) (*User, error) {
	grpcClient, err := grpc.Dial(config.grpcPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &User{
		config: config,
		client: proto.NewChatServiceClient(grpcClient),
	}, nil
}

func (u *User) Login() error {
	login, err := u.client.Login(context.Background(), &proto.LoginRequest{Username: ""})
	if err != nil {
		return err
	}

	if login.Status != "" {
		return fmt.Errorf("unable to log in: %s", login.Status)
	}

	return nil
}

func (u *User) Subscribe() error {
	var err error
	u.sub, err = u.client.SubscribeToChannel(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (u *User) SendRoutine() {
	commandCh := make(chan *proto.Command)

	go func() {
		for cmd := range commandCh {
			if err := u.sub.Send(cmd); err != nil {
				log.Println(err)
			}
		}
	}()

	go func() {
		// fake client message sender
		c := 0
		for range time.Tick(5 * time.Second) {
			commandCh <- &proto.Command{Ask: &proto.Command_Message{Message: &proto.Message{
				Body: fmt.Sprintf("%d", c),
			}}}
			c++
		}
	}()
}

func (u *User) ReceiveRoutine() {
	for {
		recv, err := u.sub.Recv()
		if err != nil {
			log.Println(err)
		}

		switch t := recv.Tell.(type) {
		case *proto.Tell_Message:
			log.Printf("%s: %s", t.Message.Sender, t.Message.Body)
		case *proto.Tell_ChangeUserStatus:
			log.Print(t.ChangeUserStatus.UserId, t.ChangeUserStatus.Status)
		}
	}
}
