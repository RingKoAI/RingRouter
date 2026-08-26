package model

import (
	"time"
)

// User represents a registered user.
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;size:64"`
	Password  string    `json:"-" gorm:"size:256"`
	Role      string    `json:"role" gorm:"size:16;default:user"` // admin, user
	Quota     int64     `json:"quota" gorm:"default:0"`            // remaining tokens
	Status    string    `json:"status" gorm:"size:16;default:active"` // active, disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Token represents an API key belonging to a user.
type Token struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index"`
	Key       string    `json:"key" gorm:"uniqueIndex;size:64"`
	Name      string    `json:"name" gorm:"size:64"`
	Status    string    `json:"status" gorm:"size:16;default:active"` // active, disabled
	Quota     int64     `json:"quota" gorm:"default:-1"`              // -1 = unlimited
	UsedQuota int64     `json:"used_quota" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `json:"-" gorm:"foreignKey:UserID"`
}

// Channel represents an upstream LLM provider configuration.
type Channel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Type      string    `json:"type" gorm:"size:32"`  // openai, claude, gemini, azure, etc.
	Name      string    `json:"name" gorm:"size:64"`
	BaseURL   string    `json:"base_url" gorm:"size:256"`
	APIKey    string    `json:"-" gorm:"size:512"`     // encrypted
	Models    string    `json:"models" gorm:"type:text"` // comma-separated model list
	Status    string    `json:"status" gorm:"size:16;default:active"` // active, disabled
	Priority  int       `json:"priority" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Log represents a request log entry.
type Log struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"index"`
	TokenID      uint      `json:"token_id" gorm:"index"`
	ChannelID    uint      `json:"channel_id" gorm:"index"`
	ModelName    string    `json:"model_name" gorm:"size:64"`
	PromptTokens int       `json:"prompt_tokens" gorm:"default:0"`
	CompTokens   int       `json:"completion_tokens" gorm:"default:0"`
	Quota        int       `json:"quota" gorm:"default:0"`
	Status       string    `json:"status" gorm:"size:16;default:success"` // success, failed
	ErrorMsg     string    `json:"error_msg" gorm:"type:text"`
	IP           string    `json:"ip" gorm:"size:64"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}

// TableName overrides for MySQL compatibility.
func (Log) TableName() string {
	return "logs"
}