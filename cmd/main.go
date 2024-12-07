package main

import (
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"ryg-task-service/conf"
	"ryg-task-service/db"
	pb "ryg-task-service/gen_proto/task_service"
	"ryg-task-service/rabbit_mq"
	"ryg-task-service/service"
)

func main() {
	cnf := conf.LoadConfig()

	db.ConnectDB(cnf.DB)
	defer db.CloseDB()

	lis, err := net.Listen("tcp", cnf.RYGTaskServiceUrl)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	publisherManager := rabbit_mq.NewPublisherManager(cnf.RabbitMQConfig)

	grpcServer := grpc.NewServer()

	taskService := service.NewTaskService(db.DB)
	challengeService := service.NewChallengeService(db.DB, publisherManager.GenericEmailQueuePublisher)
	taskService.ChallengeSvs = challengeService
	challengeService.TaskSvs = taskService

	pb.RegisterTaskServiceServer(grpcServer, taskService)
	pb.RegisterChallengeServiceServer(grpcServer, challengeService)

	fmt.Printf("User Microservice is running on port %v...", cnf.RYGTaskServiceUrl)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
