package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

type StatsHandler struct {
	db *gorm.DB
}

func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

// CalculateCapacityAndSpots calculates total capacity and remaining spots according to the auto-doubling formula.
// Base capacity C0 = 25. If active_students >= capacity, capacity doubles (C = C * 2) iteratively.
// Remaining spots = capacity - active_students.
func CalculateCapacityAndSpots(activeStudents int64) (totalCapacity int64, spotsLeft int64) {
	if activeStudents < 0 {
		activeStudents = 0
	}
	capacity := int64(25)
	for activeStudents >= capacity {
		capacity *= 2
	}
	spots := capacity - activeStudents
	return capacity, spots
}

type PlacesStatsResponse struct {
	TotalCapacity  int64 `json:"total_capacity"`
	ActiveStudents int64 `json:"active_students"`
	SpotsLeft      int64 `json:"spots_left"`
}

// GetPlacesStats handles public GET /api/v1/stats/places
func (h *StatsHandler) GetPlacesStats(c *gin.Context) {
	var activeStudents int64
	if h.db != nil {
		h.db.Model(&models.User{}).Where("has_access = true AND role != ?", "admin").Count(&activeStudents)
	}

	totalCapacity, spotsLeft := CalculateCapacityAndSpots(activeStudents)

	c.JSON(http.StatusOK, PlacesStatsResponse{
		TotalCapacity:  totalCapacity,
		ActiveStudents: activeStudents,
		SpotsLeft:      spotsLeft,
	})
}
