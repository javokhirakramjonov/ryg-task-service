package service

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	pb "ryg-task-service/gen_proto/task_service"
	"ryg-task-service/model"
)

type TaskService struct {
	db           *gorm.DB
	challengeSvs ChallengeService
	pb.UnimplementedTaskServiceServer
}

func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{
		db:           db,
		challengeSvs: *NewChallengeService(db),
	}
}

func (s *TaskService) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.Task, error) {
	if err := s.challengeSvs.ValidateChallengeBelongsToUser(req.ChallengeId, req.UserId); err != nil {
		return nil, err
	}

	var resp *pb.Task

	err := s.db.Transaction(func(tx *gorm.DB) error {
		task := &model.Task{
			Title:       req.Title,
			Description: req.Description,
			ChallengeID: req.ChallengeId,
		}

		if err := s.db.WithContext(context.Background()).Create(&task).Error; err != nil {
			return err
		}

		resp = &pb.Task{
			Id:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			ChallengeId: task.ChallengeID,
		}

		challenge, err := s.challengeSvs.GetChallenge(ctx, &pb.GetChallengeRequest{Id: task.ChallengeID, UserId: req.UserId})

		if err != nil {
			return err
		}

		for date := challenge.StartDate.AsTime(); !date.After(challenge.EndDate.AsTime()); date = date.AddDate(0, 0, 1) {
			taskAndStatus := &model.TaskAndStatus{
				TaskID: task.ID,
				Date:   date,
				Status: model.TaskStatusNotStarted,
			}

			if err := s.db.WithContext(context.Background()).Create(&taskAndStatus).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *TaskService) CreateTasks(ctx context.Context, req *pb.CreateTasksRequest) (*pb.TaskList, error) {
	createdTasks := make([]*pb.Task, 0)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, taskReq := range req.TaskRequest {
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
		Task: createdTasks,
	}

	return resp, nil
}

func (s *TaskService) GetTasksByChallengeId(ctx context.Context, req *pb.GetTasksByChallengeIdRequest) (*pb.TaskList, error) {
	if err := s.challengeSvs.ValidateChallengeBelongsToUser(req.ChallengeId, req.UserId); err != nil {
		return nil, err
	}

	var tasks []model.Task

	if err := s.db.WithContext(context.Background()).Where("challenge_id = ?", req.ChallengeId).Find(&tasks).Error; err != nil {
		return nil, err
	}

	resp := &pb.TaskList{
		Task: make([]*pb.Task, 0),
	}

	for _, task := range tasks {
		resp.Task = append(resp.Task, &pb.Task{
			Id:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			ChallengeId: task.ChallengeID,
		})
	}

	return resp, nil
}

func (s *TaskService) GetTaskById(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	if err := s.ValidateTaskBelongsToUser(req.Id, req.UserId); err != nil {
		return nil, err
	}

	var task model.Task

	if err := s.db.WithContext(context.Background()).First(&task, req.Id).Error; err != nil {
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

func (s *TaskService) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.Task, error) {
	if err := s.ValidateTaskBelongsToUser(req.Id, req.UserId); err != nil {
		return nil, err
	}

	var task model.Task

	if err := s.db.WithContext(context.Background()).First(&task, req.Id).Error; err != nil {
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

func (s *TaskService) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*emptypb.Empty, error) {
	if err := s.ValidateTaskBelongsToUser(req.Id, req.UserId); err != nil {
		return nil, err
	}

	var task model.Task

	if err := s.db.WithContext(context.Background()).First(&task, req.Id).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(context.Background()).Delete(&task).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TaskService) ValidateTaskBelongsToUser(taskId, userId int32) error {
	var task model.Task
	if err := s.db.First(&task, taskId).Error; err != nil {
		return err
	}

	if err := s.challengeSvs.ValidateChallengeBelongsToUser(task.ChallengeID, userId); err != nil {
		return err
	}

	return nil
}
