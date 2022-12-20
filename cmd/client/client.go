package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jrwalls/chat-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

type (
	user struct {
		name string
	}
)

var (
	currentUser user

	config struct {
		grpcPort string
	}

	chatClient proto.ChatServiceClient
)

func setEnvironmentFlags() {
	flag.StringVar(&config.grpcPort, "grpcPort", ":8080", "")
}

func startGrpcClient() {
	grpcClient, err := grpc.Dial(config.grpcPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	chatClient = proto.NewChatServiceClient(grpcClient)

	selectUsername()

	chCmd := make(chan *proto.Command)
	err = subscribeToChannel(chCmd)
	if err != nil {
		log.Fatal(err)
	}
}

func subscribeToChannel(chCmd chan *proto.Command) error {
	sub, err := chatClient.SubscribeToChannel(context.Background())
	if err != nil {
		return err
	}

	go func() {
		// sending go routine
		for cmd := range chCmd {
			err := sub.Send(cmd)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	go func() {
		// fake client message sender
		c := 0
		for range time.Tick(5 * time.Second) {
			chCmd <- &proto.Command{Ask: &proto.Command_Message{Message: &proto.Message{
				Body: fmt.Sprintf("%d", c),
			}}}
			c++
		}
	}()

	for {
		recv, err := sub.Recv()
		if err != nil {
			return err
		}
		switch t := recv.Tell.(type) {
		case *proto.Tell_Message:
			log.Printf("%s: %s", t.Message.Sender, t.Message.Body)
		case *proto.Tell_ChangeUserStatus:
			log.Print(t.ChangeUserStatus.UserId, t.ChangeUserStatus.Status)
		}
	}
}

func selectUsername() {
	var username string
	for {
		log.Print("username: ")
		_, err := fmt.Scanln(&username)
		if err != nil {
			log.Fatal(err)
		}

		login, err := chatClient.Login(context.Background(), &proto.LoginRequest{Username: username})
		if err != nil {
			log.Fatal(err)
		}

		if login.Status != "" {
			log.Fatalf("unable to log in: %s", login.Status)
		}

		currentUser.name = username
		log.Println("connected")
		return
	}
}

func main() {
	setEnvironmentFlags()
	flag.Parse()
	startGrpcClient()
}
