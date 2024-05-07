package service

import (
	datav1 "github.com/bxxf/znvo-backend/gen/api/data/v1"
	"github.com/bxxf/znvo-backend/internal/database"
	"github.com/bxxf/znvo-backend/internal/logger"
)

type DataService struct {
	logger   *logger.LoggerInstance
	database *database.Database
}

func NewDataService(logger *logger.LoggerInstance, database *database.Database) *DataService {
	return &DataService{
		logger:   logger,
		database: database,
	}
}

func (s *DataService) ShareData(data string, sender string, receiver string) error {
	err := s.database.UploadSharedData(sender, receiver, data)
	if err != nil {
		s.logger.Error("could not upload data: %v", err)
	}
	return err
}

func (s *DataService) GetSharedData(userId string) []*datav1.SharedDataItem {
	data, err := s.database.GetSharedData(userId)
	if err != nil {
		s.logger.Error("could not get shared data: %v", err)
	}

	return data
}
