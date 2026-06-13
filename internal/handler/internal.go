package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/service"
)

type InternalHandler struct {
	contacts *service.ContactService
}

func NewInternalHandler(contacts *service.ContactService) *InternalHandler {
	return &InternalHandler{contacts: contacts}
}

// FrequentContactCheck 查询发件人是否为常用联系人（给 inbox-svc 调用）
func (h *InternalHandler) FrequentContactCheck(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query param required"})
		return
	}

	contact, err := h.contacts.FindByEmail(userID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if contact == nil {
		c.JSON(http.StatusOK, gin.H{
			"is_frequent": false,
			"contact":     nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_frequent": contact.IsFrequent,
		"contact":     contact,
	})
}

// parseUUID is a helper
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
