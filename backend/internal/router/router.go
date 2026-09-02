package router

import (
	"context"
	"net/http"
	"time"

	"ai-ats-platform/backend/internal/handler"
	"ai-ats-platform/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Pool              *pgxpool.Pool
	AuthHandler       *handler.AuthHandler
	JobHandler        *handler.JobHandler
	CandidateHandler  *handler.CandidateHandler
	ResumeHandler     *handler.ResumeHandler
	EnterpriseHandler *handler.EnterpriseHandler
	LLMHandler        *handler.LLMHandler
	AssistantHandler  *handler.AssistantHandler
	AuthMiddleware    *middleware.AuthMiddleware
}

func Setup(deps Dependencies) *gin.Engine {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		status := "healthy"
		dbStatus := "connected"
		httpStatus := http.StatusOK

		if err := deps.Pool.Ping(ctx); err != nil {
			status = "degraded"
			dbStatus = "disconnected"
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, gin.H{
			"status":   status,
			"database": dbStatus,
			"message":  "AI ATS Backend is running 🚀",
		})
	})

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/signup", deps.AuthHandler.Signup)
			auth.POST("/login", deps.AuthHandler.Login)
			auth.GET("/me", deps.AuthMiddleware.RequireAuth(), deps.AuthHandler.Me)
		}

		jobs := v1.Group("/jobs")
		jobs.Use(deps.AuthMiddleware.RequireAuth())
		{
			jobs.POST("", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.JobHandler.Create)
			jobs.GET("", deps.JobHandler.List)
			jobs.GET("/:id/candidates", deps.CandidateHandler.ListByJob)
			jobs.GET("/:id/semantic-matches", deps.JobHandler.SemanticMatches)
			jobs.POST("/:id/ai-assistant", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.JobHandler.AIAssistant)
			jobs.GET("/:id", deps.JobHandler.GetByID)
			jobs.PUT("/:id", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.JobHandler.Update)
			jobs.DELETE("/:id", deps.AuthMiddleware.RequireRoles("admin"), deps.JobHandler.Delete)
		}

		candidates := v1.Group("/candidates")
		candidates.Use(deps.AuthMiddleware.RequireAuth())
		{
			candidates.POST("", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.CandidateHandler.Create)
			candidates.GET("", deps.CandidateHandler.List)
			candidates.GET("/:id", deps.CandidateHandler.GetByID)
			candidates.PUT("/:id", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.CandidateHandler.Update)
			candidates.DELETE("/:id", deps.AuthMiddleware.RequireRoles("admin", "recruiter"), deps.CandidateHandler.Delete)
			candidates.GET("/:id/notes", deps.CandidateHandler.ListNotes)
			candidates.POST("/:id/notes", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager", "interviewer"), deps.CandidateHandler.CreateNote)
			candidates.DELETE("/:id/notes/:noteId", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.CandidateHandler.DeleteNote)
			candidates.GET("/:id/timeline", deps.CandidateHandler.Timeline)
			candidates.GET("/:id/comments", deps.EnterpriseHandler.ListComments)
			candidates.POST("/:id/comments", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager", "interviewer"), deps.EnterpriseHandler.CreateComment)
			candidates.POST("/:id/assign", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.EnterpriseHandler.AssignCandidate)
		}

		resumes := v1.Group("/resumes")
		resumes.Use(deps.AuthMiddleware.RequireAuth())
		{
			resumes.POST("/upload", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.ResumeHandler.Upload)
			resumes.GET("/:id/file", deps.ResumeHandler.Download)
			resumes.POST("/:id/attach", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.ResumeHandler.Attach)
		}

		if deps.LLMHandler != nil {
			llmAPI := v1.Group("/llm")
			llmAPI.POST("/gemini", deps.LLMHandler.GeminiFallback)
		}

		if deps.AssistantHandler != nil {
			assistant := v1.Group("/assistant")
			assistant.Use(deps.AuthMiddleware.RequireAuth())
			{
				assistant.POST("/chat", deps.AssistantHandler.Chat)
			}
		}

		if deps.EnterpriseHandler != nil {
			analytics := v1.Group("/analytics")
			analytics.Use(deps.AuthMiddleware.RequireAuth())
			{
				analytics.GET("/overview", deps.EnterpriseHandler.Overview)
			}

			settings := v1.Group("/settings")
			settings.Use(deps.AuthMiddleware.RequireAuth())
			{
				settings.GET("/ai", deps.EnterpriseHandler.GetAISettings)
				settings.PUT("/ai", deps.AuthMiddleware.RequireRoles("admin"), deps.EnterpriseHandler.UpdateAISettings)
			}

			notifications := v1.Group("/notifications")
			notifications.Use(deps.AuthMiddleware.RequireAuth())
			{
				notifications.GET("", deps.EnterpriseHandler.ListNotifications)
				notifications.POST("/read-all", deps.EnterpriseHandler.MarkAllNotificationsRead)
				notifications.POST("/:id/read", deps.EnterpriseHandler.MarkNotificationRead)
			}

			audit := v1.Group("/audit-logs")
			audit.Use(deps.AuthMiddleware.RequireAuth(), deps.AuthMiddleware.RequireRoles("admin", "hiring_manager"))
			{
				audit.GET("", deps.EnterpriseHandler.ListAuditLogs)
			}

			interviews := v1.Group("/interviews")
			interviews.Use(deps.AuthMiddleware.RequireAuth())
			{
				interviews.GET("", deps.EnterpriseHandler.ListInterviews)
				interviews.POST("", deps.AuthMiddleware.RequireRoles("admin", "recruiter", "hiring_manager"), deps.EnterpriseHandler.CreateInterview)
			}
		}
	}

	return router
}
