package repositories

import (
	"context"
	"errors"

	"gateway/internal/models"

	"gorm.io/gorm"
)

var ErrRouteNotFound = errors.New("route not found")

type RouteRepository interface {
	Create(ctx context.Context, route *models.Route) error
	List(ctx context.Context) ([]models.Route, error)
	ListActive(ctx context.Context) ([]models.Route, error)
	GetByID(ctx context.Context, id uint) (*models.Route, error)
	Update(ctx context.Context, route *models.Route) error
	Delete(ctx context.Context, id uint) error
}

type GormRouteRepository struct {
	db *gorm.DB
}

func NewGormRouteRepository(db *gorm.DB) *GormRouteRepository {
	return &GormRouteRepository{db: db}
}

func (r *GormRouteRepository) Create(ctx context.Context, route *models.Route) error {
	return r.db.WithContext(ctx).Create(route).Error
}

func (r *GormRouteRepository) List(ctx context.Context) ([]models.Route, error) {
	var routes []models.Route
	err := r.db.WithContext(ctx).Order("path asc").Find(&routes).Error
	return routes, err
}

func (r *GormRouteRepository) ListActive(ctx context.Context) ([]models.Route, error) {
	var routes []models.Route
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("path asc").Find(&routes).Error
	return routes, err
}

func (r *GormRouteRepository) GetByID(ctx context.Context, id uint) (*models.Route, error) {
	var route models.Route
	err := r.db.WithContext(ctx).First(&route, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRouteNotFound
	}
	if err != nil {
		return nil, err
	}
	return &route, nil
}

func (r *GormRouteRepository) Update(ctx context.Context, route *models.Route) error {
	return r.db.WithContext(ctx).Save(route).Error
}

func (r *GormRouteRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Route{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRouteNotFound
	}
	return nil
}
