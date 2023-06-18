package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"grpc-chat/chatpb"

	"google.golang.org/grpc"
	"gorm.io/gorm"
	// "google.golang.org/grpc"
	// "gorm.io/driver/postgres"
	// "gorm.io/gorm"
)

var DB *gorm.DB
var err error

type chatServiceServer struct {
	chatpb.UnimplementedChatServiceServer
	mu      sync.Mutex
	channel map[string][]chan *chatpb.Message
}

func (s *chatServiceServer) JoinChannel(ch *chatpb.Channel, msgStream chatpb.ChatService_JoinChannelServer) error {

	msgChannel := make(chan *chatpb.Message)
	s.channel[ch.Name] = append(s.channel[ch.Name], msgChannel)

	for {
		select {
		case <-msgStream.Context().Done():
			return nil
		case msg := <-msgChannel:
			fmt.Printf("Got Message %v \n", msg)
			msgStream.Send(msg)
		}
	}
}

func (s *chatServiceServer) SendMessage(msgStream chatpb.ChatService_SendMessageServer) error {
	msg, err := msgStream.Recv()

	if err == io.EOF {
		return nil
	}

	if err != nil {
		return err
	}

	ack := chatpb.MessageAck{Status: "SENT"} //ack 보냄 
	msgStream.SendAndClose(&ack)

	go func() {
		streams := s.channel[msg.Channel.Name]
		for _, msgChan := range streams {
			msgChan <- msg
		}
	}()

	return nil
}

func newServer() *chatServiceServer {
	s := &chatServiceServer{
		channel: make(map[string][]chan *chatpb.Message),
	}
	fmt.Println(s)
	return s
}

// func DatabaseConnection() {
// 	host := "localhost"
// 	port := "5432"
// 	dbName := "postgres"
// 	dbUser := "postgres"
// 	password := "pass1234"
// 	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
// 		host,
// 		port,
// 		dbUser,
// 		dbName,
// 		password,
// 	)

// 	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
// 	// DB.AutoMigrate(Movie{})
// 	if err != nil {
// 		log.Fatal("Error connecting to the database...", err)
// 	}
// 	fmt.Println("Database connection successful...")
//  }

func main() {
	fmt.Println("============ SERVER CHAT APP ============")
	lis, err := net.Listen("tcp", "localhost:5400")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// DatabaseConnection() //database

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	chatpb.RegisterChatServiceServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}
