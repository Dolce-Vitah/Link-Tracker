package ormrepo

import "time"

type ChatModel struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	ChatID    int64     `gorm:"uniqueIndex;column:chat_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (ChatModel) TableName() string { return "chats" }

type LinkModel struct {
	ID            int64      `gorm:"primaryKey;column:id"`
	URL           string     `gorm:"uniqueIndex;column:url"`
	LastUpdated   *time.Time `gorm:"column:last_updated"`
	LastCheckedAt *time.Time `gorm:"column:last_checked_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (LinkModel) TableName() string { return "links" }

type ChatLinkModel struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	ChatID    int64     `gorm:"column:chat_id;uniqueIndex:idx_chat_link"`
	LinkID    int64     `gorm:"column:link_id;uniqueIndex:idx_chat_link"`
	Filters   []byte    `gorm:"column:filters"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (ChatLinkModel) TableName() string { return "chat_links" }

type TagModel struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	ChatID    int64     `gorm:"column:chat_id;uniqueIndex:idx_chat_tag"`
	Name      string    `gorm:"column:name;uniqueIndex:idx_chat_tag"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (TagModel) TableName() string { return "tags" }

type ChatLinkTagModel struct {
	ChatLinkID int64 `gorm:"column:chat_link_id;primaryKey"`
	TagID      int64 `gorm:"column:tag_id;primaryKey"`
	ChatID     int64 `gorm:"column:chat_id"`
}

func (ChatLinkTagModel) TableName() string { return "chat_link_tags" }
