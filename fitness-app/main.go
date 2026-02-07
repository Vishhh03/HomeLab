package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Workout struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Exercise    string    `json:"exercise" form:"exercise" binding:"required"`
	Reps        int       `json:"reps" form:"reps" binding:"required"`
	Weight      float64   `json:"weight" form:"weight" binding:"required"`
	RPE         int       `json:"rpe" form:"rpe"`                // 1-10 Intensity
	Tempo       string    `json:"tempo" form:"tempo"`            // e.g., "3-0-1"
	MuscleGroup string    `json:"muscle_group" form:"muscle_group"` // e.g., "Chest", "Back"
	Equipment   string    `json:"equipment" form:"equipment"`    // "Dumbbell", "Machine"
	IsFailure   bool      `json:"is_failure" form:"is_failure"`  // HIT Focus
	CreatedAt   time.Time `json:"timestamp"`
}

type BodyMetrics struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	ShoulderCircumference float64   `json:"shoulder_circumference" form:"shoulder"`
	WaistCircumference    float64   `json:"waist_circumference" form:"waist"`
	ChestCircumference    float64   `json:"chest_circumference" form:"chest"`
	CreatedAt            time.Time `json:"timestamp"`
}

var DB *gorm.DB

func initDatabase() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"),
		os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database!")
	}
	// Migrate the schema
	DB.AutoMigrate(&Workout{}, &BodyMetrics{})
}

func main() {
	initDatabase()
	r := gin.Default()

	// Load templates
	r.LoadHTMLFiles("index.html")
	r.Static("/static", "./static")

	// UI Route
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "database connected & lifting"})
	})

	// Combined API/HTMX Workout Route
	r.POST("/api/v1/workout", func(c *gin.Context) {
		var workout Workout
		
		// .ShouldBind detects if it's JSON or Form data automatically!
		if err := c.ShouldBind(&workout); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		DB.Create(&workout)

		// Check if request is from HTMX
		if c.GetHeader("HX-Request") == "true" {
			htmlSnippet := fmt.Sprintf(`
				<div class="p-3 bg-slate-700 rounded border-l-4 border-green-500 shadow-sm animate-pulse">
					<span class="font-bold text-blue-400">%s</span>: %d reps @ %.1fkg
				</div>`, workout.Exercise, workout.Reps, workout.Weight)
			c.Writer.Header().Set("Content-Type", "text/html")
			c.String(http.StatusCreated, htmlSnippet)
			return
		}

		// Otherwise, return JSON for standard API users
		c.JSON(http.StatusCreated, workout)
	})

	// Get All Workouts
	r.GET("/api/v1/workouts", func(c *gin.Context) {
		var workouts []Workout
		DB.Order("created_at desc").Find(&workouts)
		
		// If HTMX is requesting the list (initial load)
		if c.GetHeader("HX-Request") == "true" {
			var html string
			for _, w := range workouts {
				// Simple HIT Intensity indicator
				intensityBadge := ""
				if w.IsFailure {
					intensityBadge = "🔥 HIT"
				}
				html += fmt.Sprintf(`
					<div class="p-3 bg-slate-700 rounded border-l-4 border-blue-500 mb-2">
						<div class="flex justify-between items-center">
							<span class="font-bold text-lg">%s</span>
							<span class="text-xs font-bold text-red-500">%s</span>
						</div>
						<div class="text-sm text-slate-300">
							%d reps @ %.1fkg (RPE: %d)
						</div>
					</div>`, w.Exercise, intensityBadge, w.Reps, w.Weight, w.RPE)
			}
			c.Writer.Header().Set("Content-Type", "text/html")
			c.String(http.StatusOK, html)
			return
		}
		c.JSON(http.StatusOK, workouts)
	})

	// Get Target for Exercise (Progressive Overload Logic)
	r.GET("/api/v1/target", func(c *gin.Context) {
		exercise := c.Query("exercise")
		var lastWorkout Workout
		
		// Find last log for this exercise
		if result := DB.Where("exercise = ?", exercise).Order("created_at desc").First(&lastWorkout); result.Error != nil {
			c.JSON(http.StatusOK, gin.H{"weight": 0, "reps": 0, "message": "New Exercise"})
			return
		}

		// Progressive Overload Algorithm (Simple HIT)
		targetWeight := lastWorkout.Weight
		targetReps := lastWorkout.Reps

		// If last set was failure and reps > 8, increase weight by 2.5kg
		if lastWorkout.IsFailure && lastWorkout.Reps >= 8 {
			targetWeight += 2.5
		} else {
			// Otherwise try to add 1 rep
			targetReps += 1
		}

		c.JSON(http.StatusOK, gin.H{
			"weight": targetWeight,
			"reps": targetReps,
			"message": fmt.Sprintf("Last: %.1fkg x %d", lastWorkout.Weight, lastWorkout.Reps),
		})
	})

	// Log Body Metrics
	r.POST("/api/v1/metrics", func(c *gin.Context) {
		var metrics BodyMetrics
		if err := c.ShouldBind(&metrics); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		metrics.CreatedAt = time.Now()
		DB.Create(&metrics)
		c.Status(http.StatusCreated)
	})

	// Get Body Metrics for Chart
	r.GET("/api/v1/metrics", func(c *gin.Context) {
		var metrics []BodyMetrics
		DB.Order("created_at asc").Find(&metrics) // Ascending for charts
		c.JSON(http.StatusOK, metrics)
	})

	r.Run(":8081")
}
