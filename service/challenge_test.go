package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	pb "ryg-task-service/gen_proto/task_service"
	"ryg-task-service/model"
)

var testDb *gorm.DB

func setupDatabase() {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:13-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	host, _ := postgresContainer.Host(context.Background())
	port, _ := postgresContainer.MappedPort(context.Background(), "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=user password=password dbname=testdb sslmode=disable", host, port.Port())

	for i := 0; i < 5; i++ {
		testDb, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if testDb == nil {
		log.Fatalf("failed to connect to database after multiple attempts")
	}

	if err := testDb.AutoMigrate(&model.Challenge{}, &model.Task{}, &model.TaskAndStatus{}); err != nil {
		log.Fatalf("failed to migrate database: %s", err)
	}
}

func TestMain(m *testing.M) {
	setupDatabase()
	defer testDb.Exec("DROP TABLE challenges;")
	defer testDb.Exec("DROP TABLE tasks;")
	defer testDb.Exec("DROP TABLE task_and_status;")

	m.Run()
}

func clearDatabase() {
	tables, _ := testDb.Migrator().GetTables()
	for _, table := range tables {
		testDb.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE;", table))
	}
}

var createChallengeRequest = &pb.CreateChallengeRequest{
	Title:       "Test Challenge",
	Description: "Test Description",
	StartDate:   timestamppb.New(time.Now()),
	EndDate:     timestamppb.New(time.Now().AddDate(0, 1, 0)),
	UserId:      1,
}

func TestCreateChallenge(t *testing.T) {
	defer clearDatabase()
	challengeService := NewChallengeService(testDb)

	req := createChallengeRequest

	resp, err := challengeService.CreateChallenge(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, req.Title, resp.Title)
	assert.Equal(t, req.Description, resp.Description)
	assert.Equal(t, req.UserId, resp.UserId)
}

func TestGetChallenge(t *testing.T) {
	defer clearDatabase()
	challengeService := NewChallengeService(testDb)

	challenge, err := challengeService.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	req := &pb.GetChallengeRequest{Id: challenge.GetId(), UserId: challenge.GetUserId()}

	resp, err := challengeService.GetChallenge(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, challenge.GetId(), resp.Id)
	assert.Equal(t, challenge.Title, resp.Title)
	assert.Equal(t, challenge.Description, resp.Description)
}

func TestUpdateChallenge(t *testing.T) {
	defer clearDatabase()
	challengeService := &ChallengeService{db: testDb}

	challenge, err := challengeService.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	req := &pb.UpdateChallengeRequest{
		Id:          challenge.GetId(),
		Title:       "Updated Challenge",
		Description: "Updated Description",
		StartDate:   timestamppb.New(time.Now()),
		EndDate:     timestamppb.New(time.Now().AddDate(0, 2, 0)),
		UserId:      challenge.GetUserId(),
	}

	resp, err := challengeService.UpdateChallenge(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, req.Title, resp.Title)
	assert.Equal(t, req.Description, resp.Description)
}

func TestDeleteChallenge(t *testing.T) {
	defer clearDatabase()
	challengeService := &ChallengeService{db: testDb}

	challenge, err := challengeService.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	req := &pb.DeleteChallengeRequest{Id: challenge.GetId(), UserId: challenge.GetUserId()}

	resp, err := challengeService.DeleteChallenge(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, &emptypb.Empty{}, resp)

	var deletedChallenge model.Challenge
	result := testDb.First(&deletedChallenge, challenge.GetId())
	assert.Error(t, result.Error)
	assert.True(t, errors.Is(result.Error, gorm.ErrRecordNotFound))
}
