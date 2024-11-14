package service

import (
	"context"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	pb "ryg-task-service/gen_proto/task_service"
	"ryg-task-service/model"
)

type ChallengeService struct {
	db *gorm.DB
	pb.UnimplementedChallengeServiceServer
}

func NewChallengeService(db *gorm.DB) *ChallengeService {
	return &ChallengeService{
		db: db,
	}
}

func (s *ChallengeService) CreateChallenge(ctx context.Context, req *pb.CreateChallengeRequest) (*pb.Challenge, error) {
	challenge := &model.Challenge{
		Title:       req.Title,
		Description: req.Description,
		StartDate:   req.StartDate.AsTime(),
		EndDate:     req.EndDate.AsTime(),
		UserID:      req.UserId,
	}

	if err := s.db.WithContext(context.Background()).Create(&challenge).Error; err != nil {
		return nil, err
	}

	resp := &pb.Challenge{
		Id:          challenge.ID,
		Title:       challenge.Title,
		Description: challenge.Description,
		StartDate:   timestamppb.New(challenge.StartDate),
		EndDate:     timestamppb.New(challenge.EndDate),
		UserId:      challenge.UserID,
	}

	return resp, nil
}

func (s *ChallengeService) GetChallenge(ctx context.Context, req *pb.GetChallengeRequest) (*pb.Challenge, error) {
	if err := s.ValidateChallengeBelongsToUser(req.Id, req.UserId); err != nil {
		return nil, err
	}

	var challenge model.Challenge

	if err := s.db.WithContext(context.Background()).First(&challenge, req.Id).Error; err != nil {
		return nil, err
	}

	if challenge.UserID != req.UserId {
		return nil, status.Error(404, "Challenge not found")
	}

	resp := &pb.Challenge{
		Id:          challenge.ID,
		Title:       challenge.Title,
		Description: challenge.Description,
		StartDate:   timestamppb.New(challenge.StartDate),
		EndDate:     timestamppb.New(challenge.EndDate),
		UserId:      challenge.UserID,
	}

	return resp, nil
}

func (s *ChallengeService) UpdateChallenge(ctx context.Context, req *pb.UpdateChallengeRequest) (*pb.Challenge, error) {
	if err := s.ValidateChallengeBelongsToUser(req.Id, req.UserId); err != nil {
		return nil, err
	}

	var challenge model.Challenge

	if err := s.db.WithContext(context.Background()).First(&challenge, req.Id).Error; err != nil {
		return nil, err
	}

	if challenge.UserID != req.UserId {
		return nil, status.Error(404, "Challenge not found")
	}

	challenge.Title = req.Title
	challenge.Description = req.Description
	challenge.StartDate = req.StartDate.AsTime()
	challenge.EndDate = req.EndDate.AsTime()

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
	}

	return resp, nil
}

func (s *ChallengeService) DeleteChallenge(ctx context.Context, req *pb.DeleteChallengeRequest) (*emptypb.Empty, error) {
	if err := s.ValidateChallengeBelongsToUser(req.Id, req.UserId); err != nil {
		return nil, err
	}

	var challenge model.Challenge

	if err := s.db.WithContext(context.Background()).First(&challenge, req.Id).Error; err != nil {
		return nil, err
	}

	if challenge.ID != req.UserId {
		return nil, status.Error(404, "Challenge not found")
	}

	if err := s.db.WithContext(context.Background()).Delete(&challenge).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ChallengeService) ValidateChallengeBelongsToUser(challengeId, userId int32) error {
	var challenge model.Challenge

	if err := s.db.First(&challenge, challengeId).Error; err != nil {
		return err
	}

	if challenge.UserID != userId {
		return status.Error(404, "Challenge not found")
	}

	return nil
}
