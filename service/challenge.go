package service

import (
	"context"
	"fmt"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"ryg-task-service/gen_proto/email_service"
	pb "ryg-task-service/gen_proto/task_service"
	"ryg-task-service/model"
	"ryg-task-service/rabbit_mq"
	"time"
)

type ChallengeService struct {
	db                    *gorm.DB
	TaskSvs               *TaskService
	GenericEmailPublisher rabbit_mq.Publisher[*email_service.GenericEmail]
	pb.UnimplementedChallengeServiceServer
}

func NewChallengeService[P rabbit_mq.Publisher[*email_service.GenericEmail]](db *gorm.DB, genericEmailPublisher P) *ChallengeService {
	return &ChallengeService{
		db:                    db,
		GenericEmailPublisher: genericEmailPublisher,
	}
}

func (s *ChallengeService) CreateChallenge(ctx context.Context, req *pb.CreateChallengeRequest) (*pb.Challenge, error) {
	if err := validateCreateChallengeRequest(req); err != nil {
		return nil, err
	}

	challenge := &model.Challenge{
		Title:       req.Title,
		Description: req.Description,
		Status:      model.ChallengeStatusDraft,
		Days:        req.Days,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.db.WithContext(context.Background()).Create(&challenge).Error; err != nil {
			return err
		}

		challengeAndUser := &model.ChallengeAndUser{
			ChallengeID: challenge.ID,
			UserID:      req.UserId,
			UserRole:    model.ChallengeAndUserOwnerRole,
		}

		if err := s.db.WithContext(context.Background()).Create(&challengeAndUser).Error; err != nil {
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
	challenge, err := s.ValidateUserCanReadChallenge(req.Id, req.UserId)
	if err != nil {
		return nil, err
	}

	resp := &pb.Challenge{
		Id:          challenge.ID,
		Title:       challenge.Title,
		Description: challenge.Description,
		StartDate:   timestamppb.New(challenge.StartDate),
		EndDate:     timestamppb.New(challenge.EndDate),
		Status:      challenge.Status,
		Days:        challenge.Days,
	}

	return resp, nil
}

func (s *ChallengeService) GetChallengesByUserId(ctx context.Context, req *pb.GetChallengesRequest) (*pb.ChallengeList, error) {
	var challengeAndUsers []model.ChallengeAndUser

	if err := s.db.WithContext(ctx).Preload("Challenge").Where("user_id = ?", req.UserId).Find(&challengeAndUsers).Error; err != nil {
		return nil, err
	}

	resp := &pb.ChallengeList{
		Challenges: make([]*pb.Challenge, 0),
	}

	for _, challengeAndUser := range challengeAndUsers {
		challenge := challengeAndUser.Challenge
		resp.Challenges = append(resp.Challenges, &pb.Challenge{
			Id:          challenge.ID,
			Title:       challenge.Title,
			Description: challenge.Description,
			StartDate:   timestamppb.New(challenge.StartDate),
			EndDate:     timestamppb.New(challenge.EndDate),
			Status:      challenge.Status,
			Days:        challenge.Days,
		})
	}

	return resp, nil
}

func (s *ChallengeService) UpdateChallenge(ctx context.Context, req *pb.UpdateChallengeRequest) (*pb.Challenge, error) {
	challenge, err := s.ValidateChallengeOwnedByUser(req.Id, req.UserId)
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
	challenge, err := s.ValidateChallengeOwnedByUser(req.Id, req.UserId)

	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(context.Background()).Delete(&challenge).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ChallengeService) ValidateUserCanReadChallenge(challengeId, userId int64) (*model.Challenge, error) {
	challengeAndUser, err := validateChallengeBelongsToUser(challengeId, userId, s.db)

	if err != nil {
		return nil, err
	}

	return &challengeAndUser.Challenge, nil
}

func (s *ChallengeService) ValidateChallengeOwnedByUser(challengeId, userId int64) (*model.Challenge, error) {
	challengeAndUser, err := validateChallengeBelongsToUser(challengeId, userId, s.db)

	if err != nil {
		return nil, err
	}

	if challengeAndUser.UserRole != model.ChallengeAndUserOwnerRole {
		return nil, status.Error(403, "User is not the owner of the challenge")
	}

	return &challengeAndUser.Challenge, nil
}

func validateChallengeBelongsToUser(challengeId, userId int64, db *gorm.DB) (*model.ChallengeAndUser, error) {
	var challengeAndUser model.ChallengeAndUser

	if err := db.Preload("Challenge").First(&challengeAndUser, "challenge_id = ? AND user_id = ?", challengeId, userId); err.Error != nil {
		return nil, err.Error
	}

	return &challengeAndUser, nil
}

func (s *ChallengeService) StartChallenge(ctx context.Context, req *pb.StartChallengeRequest) (*pb.Challenge, error) {
	challenge, err := s.ValidateChallengeOwnedByUser(req.ChallengeId, req.UserId)
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

		participants := make([]model.ChallengeAndUser, 0)

		if err := s.db.Find(&participants, "challenge_id = ?", challenge.ID).Error; err != nil {
			return err
		}

		participantIds := make([]int64, 0)
		for _, participant := range participants {
			participantIds = append(participantIds, participant.UserID)
		}

		if err := s.TaskSvs.createTaskAndStatusesForChallenge(tx, challenge, participantIds); err != nil {
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
		Status:      challenge.Status,
		Days:        challenge.Days,
	}

	return resp, nil
}

func (s *ChallengeService) FinishChallenge(ctx context.Context, req *pb.FinishChallengeRequest) (*pb.Challenge, error) {
	challenge, err := s.ValidateChallengeOwnedByUser(req.ChallengeId, req.UserId)
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
		Status:      challenge.Status,
		Days:        challenge.Days,
	}

	return resp, nil
}

func (s *ChallengeService) AddUserToChallenge(ctx context.Context, req *pb.AddUserToChallengeRequest) (*pb.AddUserToChallengeResponse, error) {
	err := s.validateAddUserToChallengeRequest(req)

	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		challengeInvitation := &model.ChallengeInvitation{
			ChallengeID: req.ChallengeId,
			UserID:      req.UserToAddId,
			Status:      model.ChallengeInvitationStatusPending,
		}

		if err := s.db.WithContext(context.Background()).Create(&challengeInvitation).Error; err != nil {
			return err
		}

		if err := s.sendInvitationEmail(req.ChallengeId, req.UserToAddId, req.Email); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pb.AddUserToChallengeResponse{
		Message: "Invitation sent to the user",
	}, nil
}

func (s *ChallengeService) sendInvitationEmail(challengeID, userID int64, email string) error {
	token, err := GenerateJWT(userID, challengeID)

	if err != nil {
		return err
	}

	message := &email_service.GenericEmail{
		To:      email,
		Subject: "Challenge Invitation",
		Body:    fmt.Sprintf("Click the link to accept the challenge: http://rygoal.com/challenges/accept?token=%s", token),
	}

	return s.GenericEmailPublisher.Publish(message)
}

func (s *ChallengeService) validateAddUserToChallengeRequest(req *pb.AddUserToChallengeRequest) error {
	challenge, err := s.ValidateChallengeOwnedByUser(req.ChallengeId, req.UserId)

	if err != nil {
		return err
	}

	if challenge.Status == model.ChallengeStatusFinished {
		return status.Error(400, "Cannot add user to finished challenge")
	}

	if today := time.Now().Truncate(24 * time.Hour); challenge.Status == model.ChallengeStatusStarted && today.After(challenge.StartDate) {
		return status.Error(400, "Cannot add user after one day from the start date")
	}

	if _, err := s.validateUserSubscribedToChallenge(req.ChallengeId, req.UserToAddId); err == nil {
		return status.Error(400, "User already added to challenge")
	}

	return nil
}

func (s *ChallengeService) validateUserSubscribedToChallenge(challengeID, userID int64) (*model.ChallengeAndUser, error) {
	var challengeAndUser *model.ChallengeAndUser

	if err := s.db.First(&challengeAndUser, "challenge_id = ? AND user_id = ?", challengeID, userID).Error; err != nil {
		return nil, err
	}

	return challengeAndUser, nil
}

func (s *ChallengeService) SubscribeToChallenge(ctx context.Context, req *pb.SubscribeToChallengeRequest) (*pb.Challenge, error) {
	claims, err := VerifyChallengeInvitationJWT(req.Token)

	if err != nil {
		return nil, status.Error(400, "Invalid token")
	}

	if err := s.validateSubscribeToChallengeRequest(claims); err != nil {
		return nil, err
	}

	challengeAndUser := &model.ChallengeAndUser{
		ChallengeID: claims.ChallengeID,
		UserID:      claims.UserID,
		UserRole:    model.ChallengeAndUserParticipantRole,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.db.WithContext(context.Background()).Create(&challengeAndUser).Error; err != nil {
			return err
		}

		var challengeInvitation *model.ChallengeInvitation

		if err := s.db.WithContext(context.Background()).Preload("Challenge").First(&challengeInvitation, "challenge_id = ? AND user_id = ?", claims.ChallengeID, claims.UserID).Error; err != nil {
			return err
		}

		challengeInvitation.Status = model.ChallengeInvitationStatusAccepted

		if err := s.db.WithContext(context.Background()).Save(&challengeInvitation).Error; err != nil {
			return err
		}

		if err := s.TaskSvs.createTaskAndStatusesForChallenge(tx, &challengeInvitation.Challenge, []int64{claims.UserID}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetChallengeById(ctx, &pb.GetChallengeRequest{Id: claims.ChallengeID, UserId: claims.UserID})
}

func (s *ChallengeService) validateSubscribeToChallengeRequest(claims *ChallengeInvitationClaims) error {
	if _, err := s.validateUserSubscribedToChallenge(claims.ChallengeID, claims.UserID); err == nil {
		return status.Error(400, "User already added to challenge")
	}

	var challengeInvitation *model.ChallengeInvitation

	if err := s.db.Preload("Challenge").First(&challengeInvitation, "challenge_id = ? AND user_id = ?", claims.ChallengeID, claims.UserID).Error; err != nil {
		return status.Error(404, "Invitation not found")
	}

	if challengeInvitation.Challenge.Status == model.ChallengeStatusFinished {
		return status.Error(400, "Cannot subscribe to finished challenge")
	}

	today := time.Now().Truncate(24 * time.Hour)

	if challengeInvitation.Challenge.Status == model.ChallengeStatusStarted && today.After(challengeInvitation.Challenge.StartDate) {
		return status.Error(400, "Cannot subscribe after one day from the start date")
	}

	return nil
}

func (s *ChallengeService) UnsubscribeFromChallenge(ctx context.Context, req *pb.UnsubscribeFromChallengeRequest) (*emptypb.Empty, error) {
	if err := s.validateUnsubscribeFromChallengeRequest(req); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(context.Background()).Delete(&model.ChallengeAndUser{}, "challenge_id = ? AND user_id = ?", req.ChallengeId, req.UserId).Error; err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ChallengeService) validateUnsubscribeFromChallengeRequest(req *pb.UnsubscribeFromChallengeRequest) error {
	challengeAndUser, err := s.validateUserSubscribedToChallenge(req.ChallengeId, req.UserId)

	if err != nil {
		return err
	}

	if challengeAndUser.UserRole == model.ChallengeAndUserOwnerRole {
		return status.Error(400, "Cannot unsubscribe owner from challenge")
	}

	if challengeAndUser.Challenge.Status == model.ChallengeStatusFinished {
		return status.Error(400, "Cannot unsubscribe from finished challenge")
	}

	return nil
}
