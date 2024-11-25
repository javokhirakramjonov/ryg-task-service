package service

import (
	"context"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"log"
	pb "ryg-task-service/gen_proto/task_service"
	"ryg-task-service/model"
	"time"
)

type TaskService struct {
	db           *gorm.DB
	ChallengeSvs *ChallengeService
	pb.UnimplementedTaskServiceServer
}

func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{
		db: db,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.Task, error) {
	if _, err := s.ChallengeSvs.ValidateChallengeOwnedByUser(req.ChallengeId, req.UserId); err != nil {
		return nil, err
	}

	if err := validateCreateTaskRequest(req); err != nil {
		return nil, err
	}

	task := &model.Task{
		Title:       req.Title,
		Description: req.Description,
		ChallengeID: req.ChallengeId,
	}

	if err := s.db.WithContext(context.Background()).Create(&task).Error; err != nil {
		return nil, err
	}

	return &pb.Task{
		Id:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		ChallengeId: task.ChallengeID,
	}, nil
}

func validateCreateTaskRequest(req *pb.CreateTaskRequest) error {
	if req.Title == "" {
		return status.Error(400, "Title is required")
	}

	return nil
}

func (s *TaskService) createTaskAndStatusesForChallenge(tx *gorm.DB, challenge *model.Challenge, userId int64) error {
	tasks, err := s.GetTasksByChallengeId(context.Background(), &pb.GetTasksByChallengeIdRequest{
		ChallengeId: challenge.ID,
		UserId:      userId,
	})

	if err != nil {
		return err
	}

	for _, task := range tasks.Tasks {
		for date := challenge.StartDate; date.Before(challenge.EndDate); date = date.AddDate(0, 0, 1) {
			taskAndStatus := &model.TaskAndStatus{
				TaskID: task.Id,
				Date:   date,
				Status: model.TaskStatusNotStarted,
				UserID: userId,
			}

			if err := tx.WithContext(context.Background()).Create(&taskAndStatus).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *TaskService) CreateTasks(ctx context.Context, req *pb.CreateTasksRequest) (*pb.TaskList, error) {
	createdTasks := make([]*pb.Task, 0)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, taskReq := range req.TaskRequests {
			if err := validateCreateTaskRequest(taskReq); err != nil {
				return err
			}

			createdTask, err := s.CreateTask(ctx, taskReq)

			if err != nil {
				return err
			}

			createdTasks = append(createdTasks, createdTask)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	resp := &pb.TaskList{
		Tasks: createdTasks,
	}

	return resp, nil
}

func (s *TaskService) GetTasksByChallengeId(ctx context.Context, req *pb.GetTasksByChallengeIdRequest) (*pb.TaskList, error) {
	if _, err := s.ChallengeSvs.ValidateUserCanReadChallenge(req.ChallengeId, req.UserId); err != nil {
		return nil, err
	}

	var tasks []model.Task

	if err := s.db.WithContext(context.Background()).Where("challenge_id = ?", req.ChallengeId).Find(&tasks).Error; err != nil {
		return nil, err
	}

	resp := &pb.TaskList{
		Tasks: make([]*pb.Task, 0),
	}

	for _, task := range tasks {
		resp.Tasks = append(resp.Tasks, &pb.Task{
			Id:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			ChallengeId: task.ChallengeID,
		})
	}

	return resp, nil
}

func (s *TaskService) GetTaskById(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	task, err := s.validateUserCanReadTask(req.Id, req.ChallengeId, req.UserId)

	if err != nil {
		return nil, err
	}

	resp := &pb.Task{
		Id:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		ChallengeId: task.ChallengeID,
	}

	return resp, nil
}

func (s *TaskService) GetTasksByChallengeIdAndDate(ctx context.Context, req *pb.GetTaskByChallengeIdAndDateRequest) (*pb.TaskWithStatusList, error) {
	if _, err := s.ChallengeSvs.ValidateUserCanReadChallenge(req.ChallengeId, req.UserId); err != nil {
		return nil, err
	}

	var taskAndStatuses []model.TaskAndStatus

	err := s.db.Joins("JOIN tasks ON tasks.id = task_and_status.task_id").
		Preload("Task").
		Where("tasks.challenge_id = ? AND task_and_status.date = ?", req.ChallengeId, req.Date.AsTime()).
		Find(&taskAndStatuses).Error

	if err != nil {
		return nil, err
	}

	resp := &pb.TaskWithStatusList{
		TaskWithStatuses: make([]*pb.TaskWithStatus, 0),
	}

	for _, taskAndStatus := range taskAndStatuses {
		resp.TaskWithStatuses = append(resp.TaskWithStatuses, &pb.TaskWithStatus{
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

func (s *TaskService) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.Task, error) {
	task, err := s.validateTaskOwnedByUser(req.Id, req.ChallengeId, req.UserId)
	if err != nil {
		return nil, err
	}

	if err := validateUpdateTaskRequest(req); err != nil {
		return nil, err
	}

	task.Title = req.Title
	task.Description = req.Description

	if err := s.db.WithContext(context.Background()).Save(&task).Error; err != nil {
		return nil, err
	}

	resp := &pb.Task{
		Id:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		ChallengeId: task.ChallengeID,
	}

	return resp, nil
}

func validateUpdateTaskRequest(req *pb.UpdateTaskRequest) error {
	if req.Title == "" {
		return status.Error(400, "Title is required")
	}

	return nil
}

func (s *TaskService) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*emptypb.Empty, error) {
	task, err := s.validateTaskOwnedByUser(req.Id, req.ChallengeId, req.UserId)

	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(context.Background()).Delete(&task).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TaskService) validateTaskOwnedByUser(taskId, challengeId, userId int64) (*model.Task, error) {
	if _, err := s.ChallengeSvs.ValidateChallengeOwnedByUser(challengeId, userId); err != nil {
		return nil, err
	}

	var task model.Task
	if err := s.db.First(&task, taskId).Error; err != nil {
		return nil, err
	}

	if task.ChallengeID != challengeId {
		return nil, status.Error(404, "Task not found")
	}

	return &task, nil
}

func (s *TaskService) validateUserCanReadTask(taskId, challengeId, userId int64) (*model.Task, error) {
	if _, err := s.ChallengeSvs.ValidateUserCanReadChallenge(challengeId, userId); err != nil {
		return nil, err
	}

	var task model.Task
	if err := s.db.First(&task, taskId).Error; err != nil {
		return nil, err
	}

	if task.ChallengeID != challengeId {
		return nil, status.Error(404, "Task not found")
	}

	return &task, nil
}

func (s *TaskService) UpdateTaskStatus(ctx context.Context, req *pb.UpdateTaskStatusRequest) (*pb.TaskWithStatus, error) {
	if _, err := s.validateUserCanReadTask(req.TaskId, req.ChallengeId, req.UserId); err != nil {
		return nil, err
	}

	if err := s.validateUpdateTaskStatusRequest(req); err != nil {
		return nil, err
	}

	var taskAndStatus model.TaskAndStatus

	if err := s.db.WithContext(context.Background()).First(&taskAndStatus, "task_id = ? AND date = ? AND user_id = ?", req.TaskId, req.Date.AsTime(), req.UserId).Error; err != nil {
		return nil, err
	}

	taskAndStatus.Status = model.TaskStatus(req.Status)

	if err := s.db.WithContext(context.Background()).Save(&taskAndStatus).Error; err != nil {
		return nil, err
	}

	return &pb.TaskWithStatus{
		Task: &pb.Task{
			Id:          taskAndStatus.TaskID,
			Title:       taskAndStatus.Task.Title,
			Description: taskAndStatus.Task.Description,
			ChallengeId: taskAndStatus.Task.ChallengeID,
		},
		Date:   timestamppb.New(taskAndStatus.Date),
		Status: string(taskAndStatus.Status),
	}, nil
}

func (s *TaskService) validateUpdateTaskStatusRequest(req *pb.UpdateTaskStatusRequest) error {
	if req.Status == "" {
		return status.Error(400, "Status is required")
	}

	today := time.Now().Truncate(24 * time.Hour)
	requestDate := req.Date.AsTime().Truncate(24 * time.Hour)

	if requestDate != today {
		log.Printf("today: %v, requestDate: %v", today, requestDate)
		return status.Error(400, "Date should be today")
	}

	challenge, err := s.ChallengeSvs.ValidateUserCanReadChallenge(req.ChallengeId, req.UserId)

	if err != nil {
		return err
	}

	if challenge.Status != model.ChallengeStatusStarted {
		return status.Error(400, "Cannot update task status for not started or finished challenge")
	}

	return nil
}
