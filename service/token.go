package service

import (
	"fmt"
	jwt "github.com/golang-jwt/jwt"
	"time"
)

var jwtKey = []byte("your_secret_key")

const challengeInvitationExpirationTime = 24 * time.Hour

type ChallengeInvitationClaims struct {
	UserID      int64 `json:"user_id"`
	ChallengeID int64 `json:"challenge_id"`
	jwt.StandardClaims
}

func GenerateJWT(userID, challengeID int64) (string, error) {
	expirationTime := time.Now().Add(challengeInvitationExpirationTime)

	claims := &ChallengeInvitationClaims{
		UserID:      userID,
		ChallengeID: challengeID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func VerifyChallengeInvitationJWT(tokenStr string) (*ChallengeInvitationClaims, error) {
	claims := &ChallengeInvitationClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
