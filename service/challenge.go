package service

import (
	"context"
	"fmt"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	pb "ryg-task-service/gen_proto/task_service"
	"ryg-task-service/model"
	"time"
)

type ChallengeService struct {
	db      *gorm.DB
	taskSvs *TaskService
	pb.UnimplementedChallengeServiceServer
}

func NewChallengeService(db *gorm.DB) *ChallengeService {
	return &ChallengeService{
		db:      db,
		taskSvs: NewTaskService(db),
	}
}

func (s *ChallengeService) CreateChallenge(ctx context.Context, req *pb.CreateChallengeRequest) (*pb.Challenge, error) {
	if err := validateCreateChallengeRequest(req); err != nil {
		return nil, err
	}

	challenge := &model.Challenge{
		Title:       req.Title,
		Description: req.Description,
		UserID:      req.UserId,
		Status:      model.ChallengeStatusDraft,
		Days:        req.Days,
	}

	if err := s.db.WithContext(context.Background()).Create(&challenge).Error; err != nil {
		return nil, err
	}

	resp := &pb.Challenge{
		Id:          challenge.ID,
		Title:       challenge.Title,
		Description: challenge.Description,
		UserId:      challenge.UserID,
		Status:      challenge.Status,
		Days:        challenge.Days,
	}

	return resp, nil
}

func validateCreateChallengeRequest(req *pb.CreateChallengeRequest) error {
	if req.Title == "" {
		return status.Error(400, "Title is required")
	}

	if req.Days <= 0 {
		return status.Error(400, "Days must be greater than 0")
	}

	if req.Days > model.MaxChallengeDays {
		return status.Error(400, fmt.Sprintf("Days cannot be greater than %d", model.MaxChallengeDays))
	}

	return nil
}

func (s *ChallengeService) GetChallengeById(ctx context.Context, req *pb.GetChallengeRequest) (*pb.Challenge, error) {
	challenge, err := s.ValidateChallengeBelongsToUser(req.Id, req.UserId)
	if err != nil {
		return nil, err
	}

	resp := &pb.Challenge{
		Id:          challenge.ID,
		Title:       challenge.Title,
		Description: challenge.Description,
		StartDate:   timestamppb.New(challenge.StartDate),
		EndDate:     timestamppb.New(challenge.EndDate),
		UserId:      challenge.UserID,
		Status:      challenge.Status,
		Days:        challenge.Days,
	}

	return resp, nil
}

func (s *ChallengeService) GetChallengesByUserId(ctx context.Context, req *pb.GetChallengesRequest) (*pb.ChallengeList, error) {
	var challenges []model.Challenge

	if err := s.db.WithContext(ctx).Where("user_id = ?", req.UserId).Find(&challenges).Error; err != nil {
		return nil, err
	}

	resp := &pb.ChallengeList{
		Challenges: make([]*pb.Challenge, 0),
	}

	for _, challenge := range challenges {
		resp.Challenges = append(resp.Challenges, &pb.Challenge{
			Id:          challenge.ID,
			Title:       challenge.Title,
			Description: challenge.Description,
			StartDate:   timestamppb.New(challenge.StartDate),
			EndDate:     timestamppb.New(challenge.EndDate),
			UserId:      challenge.UserID,
			Status:      challenge.Status,
			Days:        challenge.Days,
		})
	}

	return resp, nil
}

func (s *ChallengeService) UpdateChallenge(ctx context.Context, req *pb.UpdateChallengeRequest) (*pb.Challenge, error) {
	challenge, err := s.ValidateChallengeBelongsToUser(req.Id, req.UserId)
	if err != nil {
		return nil, err
	}

	if challenge.Status != model.ChallengeStatusDraft {
		return nil, status.Error(400, "Cannot update started or finished challenge")
	}

	if err := validateUpdateChallengeRequest(req); err != nil {
		return nil, err
	}

	challenge.Title = req.Title
	challenge.Description = req.Description

	if err := s.db.WithContext(context.Background()).Save(&challenge).Error; err != nil {
		return nil, err
	}

	resp := &pb.Challenge{
		Id:          challenge.ID,
		Title:       challenge.Title,
		Description: challenge.Description,
		StartDate:   timestamppb.New(challenge.StartDate),
		EndDate:     timestamppb.New(challenge.EndDate),
		UserId:      challenge.UserID,
		Status:      challenge.Status,
	}

	return resp, nil
}

func validateUpdateChallengeRequest(req *pb.UpdateChallengeRequest) error {
	if req.Title == "" {
		return status.Error(400, "Title is required")
	}

	if req.Days <= 0 {
		return status.Error(400, "Days must be greater than 0")
	}

	if req.Days > model.MaxChallengeDays {
		return status.Error(400, fmt.Sprintf("Days cannot be greater than %d", model.MaxChallengeDays))
	}

	return nil
}

func (s *ChallengeService) DeleteChallenge(ctx context.Context, req *pb.DeleteChallengeRequest) (*emptypb.Empty, error) {
	challenge, err := s.ValidateChallengeBelongsToUser(req.Id, req.UserId)

	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(context.Background()).Delete(&challenge).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ChallengeService) ValidateChallengeBelongsToUser(challengeId, userId int64) (*model.Challenge, error) {
	var challenge model.Challenge

	if err := s.db.First(&challenge, challengeId).Error; err != nil {
		return nil, err
	}

	if challenge.UserID != userId {
		return nil, status.Error(404, "Challenge not found")
	}

	return &challenge, nil
}

func (s *ChallengeService) StartChallenge(ctx context.Context, req *pb.StartChallengeRequest) (*pb.Challenge, error) {
	challenge, err := s.ValidateChallengeBelongsToUser(req.ChallengeId, req.UserId)
	if err != nil {
		return nil, err
	}

	if challenge.Status != model.ChallengeStatusDraft {
		return nil, status.Error(400, "Cannot start started or finished challenge")
	}

	today := time.Now().Truncate(24 * time.Hour)

	challenge.StartDate = today
	challenge.EndDate = today.AddDate(0, 0, int(challenge.Days))
	challenge.Status = model.ChallengeStatusStarted

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.db.WithContext(context.Background()).Save(&challenge).Error; err != nil {
			return err
		}

		if err := s.taskSvs.createTaskAndStatusesForChallenge(tx, challenge); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	resp := &pb.Challenge{
		Id:          challenge.ID,
		Title:       challenge.Title,
		Description: challenge.Description,
		StartDate:   timestamppb.New(challenge.StartDate),
		EndDate:     timestamppb.New(challenge.EndDate),
		UserId:      challenge.UserID,
		Status:      challenge.Status,
		Days:        challenge.Days,
	}

	return resp, nil
}

func (s *ChallengeService) FinishChallenge(ctx context.Context, req *pb.FinishChallengeRequest) (*pb.Challenge, error) {
	challenge, err := s.ValidateChallengeBelongsToUser(req.ChallengeId, req.UserId)
	if err != nil {
		return nil, err
	}

	if challenge.Status != model.ChallengeStatusStarted {
		return nil, status.Error(400, "Cannot finish draft or finished challenge")
	}

	challenge.Status = model.ChallengeStatusFinished

	if err := s.db.WithContext(context.Background()).Save(&challenge).Error; err != nil {
		return nil, err
	}

	resp := &pb.Challenge{
		Id:          challenge.ID,
		Title:       challenge.Title,
		Description: challenge.Description,
		StartDate:   timestamppb.New(challenge.StartDate),
		EndDate:     timestamppb.New(challenge.EndDate),
		UserId:      challenge.UserID,
		Status:      challenge.Status,
		Days:        challenge.Days,
	}

	return resp, nil
}
