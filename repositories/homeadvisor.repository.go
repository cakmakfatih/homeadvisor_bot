package repositories

import (
	"errors"
	"homeadvisorbot/core"
	"homeadvisorbot/models"
	"log"

	"gorm.io/gorm"
)

type HomeadvisorRepository struct {
	db *core.Database
}

func NewHomeadvisorRepository(db *core.Database) *HomeadvisorRepository {
	return &HomeadvisorRepository{
		db: db,
	}
}

func (r *HomeadvisorRepository) DoesItExist(url string) (error, bool) {
	var m *models.HomeadvisorListingModel

	result := (*r.db).Db.Where("url = ?", url).First(&m)

	if result.Error == nil {
		log.Println(url + " already exists in the database.")
		return nil, true
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, false
	} else {
		log.Println(result.Error)
		return result.Error, false
	}
}

func (r *HomeadvisorRepository) Get(url string) (*models.HomeadvisorListingModel, error) {
	var m *models.HomeadvisorListingModel

	result := (*r.db).Db.Where("url = ?", url).First(&m)

	if result.Error != nil {
		return m, result.Error
	}

	return m, nil
}

func (r *HomeadvisorRepository) Create(m *models.HomeadvisorListingModel) {
	(*r.db).Db.Create(m)
}

func (r *HomeadvisorRepository) All() []models.HomeadvisorListingModel {
	var listings []models.HomeadvisorListingModel
	(*r.db).Db.Find(&listings)

	return listings
}
