package models

import (
	"time"
)

// Time is a custom time type for GraphQL
type Time time.Time

// JSON is a custom JSON type for GraphQL
type JSON map[string]interface{}

// Upload represents a file upload for GraphQL
type Upload struct {
	File     interface{} `json:"file"`
	Filename string      `json:"filename"`
	Size     int64       `json:"size"`
	MIMEType string      `json:"mimetype"`
}

// Pagination input for GraphQL queries
type PaginationInput struct {
	Limit  *int `json:"limit"`
	Offset *int `json:"offset"`
}

// Search filter for GraphQL
type SearchFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// Search sort for GraphQL
type SearchSort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}
