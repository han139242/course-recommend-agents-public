package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	assistantdto "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/dto"
	assistantservice "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// Handler exposes the authenticated assistant HTTP API.
type Handler struct {
	service *assistantservice.Service
}

// New creates an assistant handler.
func New(service *assistantservice.Service) *Handler {
	return &Handler{service: service}
}

// Chat handles the authenticated assistant chat endpoint and streams the final
// answer as SSE so the existing frontend typing UX can stay unchanged.
func (h *Handler) Chat(ctx *gin.Context) {
	if h.service == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": apperr.ErrDependencyUnavailable.Message})
		return
	}

	var request assistantdto.ChatRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apperr.ErrBadRequest.Message})
		return
	}

	response, err := h.service.Chat(ctx.Request.Context(), request)
	if err != nil {
		appErr := apperr.AsAppError(err)
		ctx.JSON(appErr.HTTPStatus, gin.H{"error": appErr.Message})
		return
	}

	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	flusher, _ := ctx.Writer.(http.Flusher)
	for _, chunk := range splitContent(response.Content, 72) {
		payload, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{
					"delta": map[string]any{
						"content": chunk,
					},
				},
			},
		})
		_, _ = ctx.Writer.Write([]byte("data: "))
		_, _ = ctx.Writer.Write(payload)
		_, _ = ctx.Writer.Write([]byte("\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}

	payload, _ := json.Marshal(map[string]any{
		"type":       "assistant_meta",
		"actions":    response.Actions,
		"agent":      response.Agent,
		"agentLabel": response.AgentLabel,
	})
	_, _ = ctx.Writer.Write([]byte("data: "))
	_, _ = ctx.Writer.Write(payload)
	_, _ = ctx.Writer.Write([]byte("\n\n"))
	if flusher != nil {
		flusher.Flush()
	}

	_, _ = ctx.Writer.Write([]byte("data: [DONE]\n\n"))
	if flusher != nil {
		flusher.Flush()
	}
}

// PublicChat handles the unauthenticated assistant endpoint for graph/page control.
func (h *Handler) PublicChat(ctx *gin.Context) {
	if h.service == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": apperr.ErrDependencyUnavailable.Message})
		return
	}

	var request assistantdto.ChatRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apperr.ErrBadRequest.Message})
		return
	}

	response, err := h.service.ChatPublic(ctx.Request.Context(), request)
	if err != nil {
		appErr := apperr.AsAppError(err)
		ctx.JSON(appErr.HTTPStatus, gin.H{"error": appErr.Message})
		return
	}

	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	flusher, _ := ctx.Writer.(http.Flusher)
	for _, chunk := range splitContent(response.Content, 72) {
		payload, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{
					"delta": map[string]any{
						"content": chunk,
					},
				},
			},
		})
		_, _ = ctx.Writer.Write([]byte("data: "))
		_, _ = ctx.Writer.Write(payload)
		_, _ = ctx.Writer.Write([]byte("\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}

	payload, _ := json.Marshal(map[string]any{
		"type":       "assistant_meta",
		"actions":    response.Actions,
		"agent":      response.Agent,
		"agentLabel": response.AgentLabel,
	})
	_, _ = ctx.Writer.Write([]byte("data: "))
	_, _ = ctx.Writer.Write(payload)
	_, _ = ctx.Writer.Write([]byte("\n\n"))
	if flusher != nil {
		flusher.Flush()
	}

	_, _ = ctx.Writer.Write([]byte("data: [DONE]\n\n"))
	if flusher != nil {
		flusher.Flush()
	}
}

// GenerateKnowledgeCard handles the personalized knowledge card generation endpoint.
func (h *Handler) GenerateKnowledgeCard(ctx *gin.Context) {
	if h.service == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": apperr.ErrDependencyUnavailable.Message})
		return
	}

	var request assistantdto.KnowledgeCardRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apperr.ErrBadRequest.Message})
		return
	}

	response, err := h.service.GenerateKnowledgeCard(ctx.Request.Context(), request)
	if err != nil {
		appErr := apperr.AsAppError(err)
		ctx.JSON(appErr.HTTPStatus, gin.H{"error": appErr.Message})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"content": response.Content, "sources": response.Sources})
}

func splitContent(content string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = 72
	}
	runes := []rune(content)
	if len(runes) == 0 {
		return []string{""}
	}

	result := make([]string, 0, (len(runes)/chunkSize)+1)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[start:end]))
	}
	return result
}
