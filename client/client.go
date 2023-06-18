package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"grpc-chat/chatpb"

	"google.golang.org/grpc"
	"gorm.io/gorm"
)

var DB *gorm.DB

var channelName = flag.String("roomname", "default", "Room name for chatting") //ex ) -roomname=ChatRoom1
var senderName = flag.String("name", "default", "User name") //ex) -name=Kay
var tcpServer = flag.String("server", ":5400", "Tcp server")

//join channel
func joinChannel(ctx context.Context, client chatpb.ChatServiceClient) {

	channel := chatpb.Channel{Name: *channelName, SendersName: *senderName}
	stream, err := client.JoinChannel(ctx, &channel)
	if err != nil {
		log.Fatalf("client.JoinChannel(ctx, &channel) throws: %v", err)
	}

	fmt.Printf("Chat Room Name: %v \n", *channelName)
	fmt.Printf("User Name: %v \n", *senderName)
	fmt.Println("======================================")
	fmt.Printf("Entered %v\n", *channelName)

	waitc := make(chan struct{})

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive message from channel joining. \nErr: %v", err) //채널 들어가는거 실패
			}

			if *senderName != in.Sender { // 메세지가 왔을 때!!
				// now := time.Now()
				fmt.Printf("%v [%v] %v \n", in.Channel.Time, in.Sender, in.Message)
				
			}
		}
	}()

	<-waitc // wait
}

func sendMessage(ctx context.Context, client chatpb.ChatServiceClient, message string) {
	
	stream, err := client.SendMessage(ctx)

	if err != nil {
		log.Printf("Cannot send message: error: %v", err)
	}

	//시간 파싱
	now := time.Now()
	customtime := now.Format(time.RFC3339)
	// fmt.Println(customtime)

	msg := chatpb.Message{
		Channel: &chatpb.Channel{
			Time:          customtime,
			Name:        *channelName,
			SendersName: *senderName},
		Message: message,
		Sender:  *senderName,
	}
	stream.Send(&msg)

	//DB 저장
	// data := chatpb.AddMessage {
	// 	Time : customtime,
	// 	ChannelName:*channelName,
	// 	Sender: *senderName,
	// 	Message: message,
	// }

	// res := DB.Create(&data)
	// if res.RowsAffected == 0 {
	// 	log.Printf("Database insert failed..")
	// 	// return nil, errors.New("Database insert failed..")
	// }
	// ack, err := stream.CloseAndRecv()
	stream.CloseAndRecv()
}

func main() {

	flag.Parse()

	fmt.Println("========= CLIENT CHAT APP ===========")
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithInsecure())

	conn, err := grpc.Dial(*tcpServer, opts...)
	if err != nil {
		log.Fatalf("Fail to dail: %v", err)
	}

	defer conn.Close()

	ctx := context.Background()
	client := chatpb.NewChatServiceClient(conn)

	go joinChannel(ctx, client)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		go sendMessage(ctx, client, scanner.Text())
	}

}
