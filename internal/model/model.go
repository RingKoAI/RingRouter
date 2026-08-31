package model

import (
	"time"
)

// User represents a registered user.
type User struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Username    string    `json:"username" gorm:"uniqueIndex;size:64"`
	Email       string    `json:"email" gorm:"index;size:256"` // empty = not set; uniqueness enforced at the application layer so multiple email-less accounts coexist
	DisplayName string    `json:"display_name" gorm:"size:64"`
	Password    string    `json:"-" gorm:"size:256"`                          // bcrypt hash, empty for externally provisioned
	Role        string    `json:"role" gorm:"size:16;default:user"`           // admin, user
	Group       string    `json:"group" gorm:"size:64;default:default;index"` // routing group
	Quota       int64     `json:"quota" gorm:"default:0"`                     // remaining points, -1 = unlimited
	UsedQuota   int64     `json:"used_quota" gorm:"default:0"`                // lifetime consumption in points
	Status      string    `json:"status" gorm:"size:16;default:active"`       // active, disabled
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Session represents an authenticated management-session. The cookie carries
// the raw token; the database stores only its SHA-256 digest so a DB leak
// cannot be replayed as a valid session.
type Session struct {
	ID        string    `json:"id" gorm:"primaryKey;size:64"` // SHA-256 hex of cookie token
	UserID    uint      `json:"user_id" gorm:"index"`         // 0 = admin-key bootstrap session
	ExpiresAt time.Time `json:"expires_at" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
}

// Token represents an API key belonging to a user.
type Token struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	UserID    uint   `json:"user_id" gorm:"index"`
	Key       string `json:"key" gorm:"uniqueIndex;size:64"`
	Name      string `json:"name" gorm:"size:64"`
	Group     string `json:"group" gorm:"size:64;default:default;index"` // routing group; mirrors user.group at creation
	Status    string `json:"status" gorm:"size:16;default:active"`       // active, disabled
	Quota     int64  `json:"quota" gorm:"default:-1"`                    // -1 = unlimited
	UsedQuota int64  `json:"used_quota" gorm:"default:0"`
	// Token-level restrictions (one-api semantics; empty = unrestricted).
	Models    string    `json:"models" gorm:"size:512"`  // comma whitelist of model names
	Subnet    string    `json:"subnet" gorm:"size:256"`  // comma CIDRs the caller IP must match
	ExpiredAt time.Time `json:"expired_at" gorm:"index"` // zero = never expires
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `json:"-" gorm:"foreignKey:UserID"`
}

// Channel represents an upstream LLM provider endpoint. The protocol field
// determines which wire format (request/response) to use when talking to
// the upstream. Any client protocol can route to any channel protocol,
// thanks to the unified dto.ChatRequest/ChatResponse intermediate layer.
type Channel struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"size:64"`
	Protocol     string    `json:"protocol" gorm:"size:32;default:openai"` // openai | openai-compatible | anthropic | gemini
	BaseURL      string    `json:"base_url" gorm:"size:256"`
	APIKey       string    `json:"-" gorm:"size:512"`              // AES-GCM sealed at rest
	Models       string    `json:"models" gorm:"type:text"`        // comma-separated model list
	ModelMapping string    `json:"model_mapping" gorm:"type:text"` // JSON: {"client_model":"upstream_model"}
	Group        string    `json:"group" gorm:"size:64;default:default;index"`
	Status       string    `json:"status" gorm:"size:16;default:active"` // active, disabled
	Priority     int       `json:"priority" gorm:"default:0"`
	Weight       int       `json:"weight" gorm:"default:0"`
	Remark       string    `json:"remark" gorm:"size:256"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
	Quota        int64     `json:"quota" gorm:"default:0"`
	ElapsedMs    int64     `json:"elapsed_ms" gorm:"default:0"`
	Status       string    `json:"status" gorm:"size:16;default:success"` // success, failed
	ErrorMsg     string    `json:"error_msg" gorm:"type:text"`
	IP           string    `json:"ip" gorm:"size:64"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}

// Option stores a persistent key/value setting (usage mode, SMTP config, ...).
type Option struct {
	Key       string    `json:"key" gorm:"primaryKey;size:64"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Code is a short-lived email verification code (password reset). The code
// itself is never stored — only its salted SHA-256 digest — so a database
// leak cannot be replayed as a usable code. Attempts bounds brute force.
type Code struct {
	ID        uint      `json:"-" gorm:"primaryKey"`
	Email     string    `json:"-" gorm:"size:256;index"`
	CodeHash  string    `json:"-" gorm:"size:128"` // hex sha256(code + salt)
	Attempts  int       `json:"-" gorm:"default:0"`
	Used      bool      `json:"-" gorm:"default:false;index"`
	ExpiresAt time.Time `json:"-" gorm:"index"`
	CreatedAt time.Time `json:"-"`
}

// Passkey stores a WebAuthn credential bound to a user. Binary fields hold
// the exact bytes produced by the authenticator so the go-webauthn library
// can verify assertions without any re-encoding.
type Passkey struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"index"`
	Name            string    `json:"name" gorm:"size:64"` // user label, e.g. "MacBook Touch ID"
	CredentialID    []byte    `json:"-" gorm:"uniqueIndex;size:1024"`
	PublicKey       []byte    `json:"-" gorm:"size:512"` // COSE
	AttestationType string    `json:"-" gorm:"size:64"`
	Transport       string    `json:"-" gorm:"size:128"` // csv: internal,hybrid,usb,nfc,ble
	AAGUID          string    `json:"-" gorm:"size:36"`
	SignCount       uint32    `json:"-" gorm:"default:0"`
	BackupEligible  bool      `json:"backup_eligible" gorm:"default:false"`
	BackupState     bool      `json:"backup_state" gorm:"default:false"`
	LastUsedAt      time.Time `json:"last_used_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ModelMeta carries the public catalogue entry for one model: vendor brand,
// description, and per-million-token prices in USD. Prices are the list
// price; the effective price per group is list × group ratio. A missing meta
// row simply means the model shows up without pricing.
type ModelMeta struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name" gorm:"uniqueIndex;size:128"`
	Vendor        string    `json:"vendor" gorm:"size:64"` // deepseek, openai, zhipu, ...
	Description   string    `json:"description" gorm:"size:512"`
	InputPrice    float64   `json:"input_price"`                          // $ / 1M tokens
	OutputPrice   float64   `json:"output_price"`                         // $ / 1M tokens
	CachePrice    float64   `json:"cache_price"`                          // $ / 1M cache-read tokens
	ContextWindow int64     `json:"context_window"`                       // tokens, 0 = unknown
	Status        string    `json:"status" gorm:"size:16;default:active"` // active, hidden
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Plan is a subscription plan: a named bundle of quota, routing group, and
// duration offered to users. Prices are stored in minor units (cents) as a
// display value only — billing integrations are out of scope for the core.
type Plan struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"uniqueIndex;size:64"`
	Description  string    `json:"description" gorm:"size:256"`
	PriceCents   int64     `json:"price_cents" gorm:"default:0"`         // display-only, 0 = free
	Quota        int64     `json:"quota" gorm:"default:0"`               // granted quota, -1 = unlimited
	Group        string    `json:"group" gorm:"size:64;index"`           // routing group (group ratio applies)
	DurationDays int       `json:"duration_days" gorm:"default:30"`      // 0 = perpetual
	Status       string    `json:"status" gorm:"size:16;default:active"` // active, disabled
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Subscription records a user being granted a plan. Plan fields are snapshotted
// at grant time so later plan edits never retroactively change active
// subscriptions; renewals re-snapshot.
type Subscription struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index"`
	PlanID    uint      `json:"plan_id" gorm:"index"`
	PlanName  string    `json:"plan_name" gorm:"size:64"`                   // snapshot
	Group     string    `json:"group" gorm:"size:64;index"`                 // snapshot
	Quota     int64     `json:"quota"`                                      // snapshot: granted quota (-1 = unlimited)
	ExpiresAt time.Time `json:"expires_at" gorm:"index"`                    // zero = perpetual
	Status    string    `json:"status" gorm:"size:16;default:active;index"` // active, expired, cancelled
	CreatedAt time.Time `json:"created_at"`
}

// Expired reports whether the subscription window has passed. Perpetual
// subscriptions (zero ExpiresAt) never expire.
func (s *Subscription) Expired() bool {
	return !s.ExpiresAt.IsZero() && time.Now().After(s.ExpiresAt)
}

// Group is a routing partition: a token is bound to one group at a time, and
// only channels whose group matches are eligible. Each group has a stable
// UUID for external references, a free-form metadata field for notes, and a
// billing ratio applied to usage from this group (1.0 = list price).
type Group struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;size:64"`
	UUID      string    `json:"uuid" gorm:"uniqueIndex;size:32"`
	Metadata  string    `json:"metadata" gorm:"type:text"`
	Ratio     float64   `json:"ratio" gorm:"default:1"` // billing multiplier: 0.8 = 20% off
	IsDefault bool      `json:"is_default" gorm:"default:false;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides for MySQL compatibility.
func (Log) TableName() string {
	return "logs"
}
