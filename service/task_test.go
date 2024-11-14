package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	pb "ryg-task-service/gen_proto/task_service"
	"ryg-task-service/model"
	"testing"
)

var createTaskRequest = &pb.CreateTaskRequest{
	Title:       "Test Task",
	Description: "Test Task Description",
	ChallengeId: 1,
	UserId:      1,
}

func TestCreateTask(t *testing.T) {
	defer clearDatabase()
	taskService := NewTaskService(testDb)

	challenge, err := taskService.challengeSvs.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	req := createTaskRequest
	req.ChallengeId = challenge.GetId()

	resp, err := taskService.CreateTask(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, req.Title, resp.Title)
	assert.Equal(t, req.Description, resp.Description)
	assert.Equal(t, req.ChallengeId, resp.ChallengeId)
}

func TestCreateTasks(t *testing.T) {
	defer clearDatabase()
	taskService := NewTaskService(testDb)

	challenge, err := taskService.challengeSvs.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	req := &pb.CreateTasksRequest{
		TaskRequest: []*pb.CreateTaskRequest{
			{Title: "Task 1", Description: "Task 1 Description", ChallengeId: challenge.GetId(), UserId: 1},
			{Title: "Task 2", Description: "Task 2 Description", ChallengeId: challenge.GetId(), UserId: 1},
		},
	}

	resp, err := taskService.CreateTasks(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, len(req.TaskRequest), len(resp.Task))

	for i, task := range req.TaskRequest {
		assert.Equal(t, task.Title, resp.Task[i].Title)
		assert.Equal(t, task.Description, resp.Task[i].Description)
		assert.Equal(t, task.ChallengeId, resp.Task[i].ChallengeId)
	}
}

func TestGetTasksByChallengeId(t *testing.T) {
	defer clearDatabase()
	taskService := NewTaskService(testDb)

	challenge, err := taskService.challengeSvs.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	taskService.CreateTask(context.Background(), &pb.CreateTaskRequest{Title: "Task 1", Description: "Task 1 Description", ChallengeId: challenge.GetId(), UserId: 1})
	taskService.CreateTask(context.Background(), &pb.CreateTaskRequest{Title: "Task 2", Description: "Task 2 Description", ChallengeId: challenge.GetId(), UserId: 1})

	req := &pb.GetTasksByChallengeIdRequest{ChallengeId: challenge.GetId(), UserId: 1}

	resp, err := taskService.GetTasksByChallengeId(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, resp.Task, 2)
}

func TestGetTaskById(t *testing.T) {
	defer clearDatabase()
	taskService := NewTaskService(testDb)

	challenge, err := taskService.challengeSvs.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	task, err := taskService.CreateTask(context.Background(), &pb.CreateTaskRequest{Title: "Test Task", Description: "Test Task Description", ChallengeId: challenge.GetId(), UserId: 1})
	assert.NoError(t, err)

	req := &pb.GetTaskRequest{Id: task.GetId(), UserId: 1}

	resp, err := taskService.GetTaskById(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, task.GetId(), resp.Id)
	assert.Equal(t, task.Title, resp.Title)
	assert.Equal(t, task.Description, resp.Description)
}

func TestUpdateTask(t *testing.T) {
	defer clearDatabase()
	taskService := NewTaskService(testDb)

	challenge, err := taskService.challengeSvs.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	task, err := taskService.CreateTask(context.Background(), &pb.CreateTaskRequest{Title: "Initial Task", Description: "Initial Description", ChallengeId: challenge.GetId(), UserId: 1})
	assert.NoError(t, err)

	req := &pb.UpdateTaskRequest{
		Id:          task.GetId(),
		Title:       "Updated Task",
		Description: "Updated Description",
		UserId:      1,
	}

	resp, err := taskService.UpdateTask(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, req.Title, resp.Title)
	assert.Equal(t, req.Description, resp.Description)
}

func (s *TaskService) TestGetTasksByChallengeIdAndDate(ctx context.Context, req *pb.GetTaskByChallengeIdAndDateRequest) (*pb.TaskWithStatusList, error) {
	if err := s.challengeSvs.ValidateChallengeBelongsToUser(req.ChallengeId, req.UserId); err != nil {
		return nil, err
	}

	var taskAndStatuses []model.TaskAndStatus

	err := s.db.Joins("Task").
		Where("Task.challenge_id = ? AND task_and_status.date = ?", req.ChallengeId, req.Date).
		Find(&taskAndStatuses).Error

	if err != nil {
		return nil, err
	}

	resp := &pb.TaskWithStatusList{
		TaskWithStatus: make([]*pb.TaskWithStatus, 0),
	}

	for _, taskAndStatus := range taskAndStatuses {
		resp.TaskWithStatus = append(resp.TaskWithStatus, &pb.TaskWithStatus{
			Task: &pb.Task{
				Id:          taskAndStatus.Task.ID,
				Title:       taskAndStatus.Task.Title,
				Description: taskAndStatus.Task.Description,
				ChallengeId: taskAndStatus.Task.ChallengeID,
			},
			Date:   timestamppb.New(taskAndStatus.Date),
			Status: string(taskAndStatus.Status),
		})
	}

	return resp, nil
}

func TestDeleteTask(t *testing.T) {
	defer clearDatabase()
	taskService := NewTaskService(testDb)

	challenge, err := taskService.challengeSvs.CreateChallenge(context.Background(), createChallengeRequest)
	assert.NoError(t, err)

	task, err := taskService.CreateTask(context.Background(), &pb.CreateTaskRequest{Title: "Task to Delete", Description: "Task Description", ChallengeId: challenge.GetId(), UserId: 1})
	assert.NoError(t, err)

	req := &pb.DeleteTaskRequest{Id: task.GetId(), UserId: 1}

	resp, err := taskService.DeleteTask(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, &emptypb.Empty{}, resp)

	var deletedTask model.Task
	result := testDb.First(&deletedTask, task.GetId())
	assert.Error(t, result.Error)
	assert.True(t, errors.Is(result.Error, gorm.ErrRecordNotFound))
}
