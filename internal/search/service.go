package search

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"gorm.io/gorm"
)

type SearchParams struct {
	Query          string   `json:"query"`
	ContentTypeIDs []uint   `json:"content_type_ids,omitempty"`
	Fields         []string `json:"fields,omitempty"`
	Status         string   `json:"status,omitempty"`
	CreatedBy      uint     `json:"created_by,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	FromDate       string   `json:"from_date,omitempty"`
	ToDate         string   `json:"to_date,omitempty"`
	Page           int      `json:"page"`
	Limit          int      `json:"limit"`
	SortBy         string   `json:"sort_by"`
	OrderBy        string   `json:"order_by"`
}

type SearchResult struct {
	Entries    []models.ContentEntry `json:"entries"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalPages int64                 `json:"total_pages"`
	Query      string                `json:"query"`
	Facets     *SearchFacets         `json:"facets,omitempty"`
}

type SearchFacets struct {
	ContentTypes map[string]int64 `json:"content_types"`
	Statuses     map[string]int64 `json:"statuses"`
	DateRange    *DateRangeFacet  `json:"date_range"`
}

type DateRangeFacet struct {
	Oldest time.Time `json:"oldest"`
	Newest time.Time `json:"newest"`
}

func FullTextSearch(params SearchParams) (*SearchResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.SortBy == "" {
		params.SortBy = "created_at"
	}
	if params.OrderBy == "" {
		params.OrderBy = "desc"
	}

	query := database.DB.Model(&models.ContentEntry{})

	if len(params.ContentTypeIDs) > 0 {
		query = query.Where("content_type_id IN ?", params.ContentTypeIDs)
	}

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if params.CreatedBy > 0 {
		query = query.Where("created_by = ?", params.CreatedBy)
	}

	if params.FromDate != "" {
		fromDate, err := time.Parse("2006-01-02", params.FromDate)
		if err == nil {
			query = query.Where("created_at >= ?", fromDate)
		}
	}
	if params.ToDate != "" {
		toDate, err := time.Parse("2006-01-02", params.ToDate)
		if err == nil {
			toDate = toDate.Add(24 * time.Hour)
			query = query.Where("created_at < ?", toDate)
		}
	}

	if params.Query != "" {
		query = applyFullTextSearch(query, params)
	}

	if len(params.Tags) > 0 {
		query = applyTagFilter(query, params.Tags)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	query = applySorting(query, params)

	offset := (params.Page - 1) * params.Limit
	query = query.Offset(offset).Limit(params.Limit)

	query = query.Preload("Creator").Preload("Updater")

	var entries []models.ContentEntry
	if err := query.Find(&entries).Error; err != nil {
		return nil, err
	}

	totalPages := total / int64(params.Limit)
	if total%int64(params.Limit) > 0 {
		totalPages++
	}

	result := &SearchResult{
		Entries:    entries,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
		Query:      params.Query,
	}

	return result, nil
}

func applyFullTextSearch(query *gorm.DB, params SearchParams) *gorm.DB {
	searchQuery := strings.TrimSpace(params.Query)
	if searchQuery == "" {
		return query
	}

	dbDialect := database.DB.Dialector.Name()

	if dbDialect == "postgres" {
		if len(params.Fields) > 0 {
			var conditions []string
			var args []any

			for _, field := range params.Fields {
				conditions = append(conditions, fmt.Sprintf("data->>'%s' ILIKE ?", field))
				args = append(args, "%"+searchQuery+"%")
			}

			whereClause := strings.Join(conditions, " OR ")
			query = query.Where(whereClause, args...)
		} else {
			tsQuery := strings.ReplaceAll(searchQuery, " ", " & ")
			query = query.Where(
				"to_tsvector('english', data::text) @@ plainto_tsquery('english', ?)",
				tsQuery,
			)
		}
	} else {
		if len(params.Fields) > 0 {
			var conditions []string
			var args []interface{}

			for _, field := range params.Fields {
				conditions = append(conditions, "json_extract(data, '$."+field+"') LIKE ?")
				args = append(args, "%"+searchQuery+"%")
			}

			whereClause := strings.Join(conditions, " OR ")
			query = query.Where(whereClause, args...)
		} else {
			query = query.Where("data LIKE ?", "%"+searchQuery+"%")
		}
	}

	return query
}

func applyTagFilter(query *gorm.DB, tags []string) *gorm.DB {
	dbDialect := database.DB.Dialector.Name()

	if dbDialect == "postgres" {
		for _, tag := range tags {
			query = query.Where("data->'tags' @> ?", fmt.Sprintf(`["%s"]`, tag))
		}
	} else {
		for _, tag := range tags {
			query = query.Where("data LIKE ?", fmt.Sprintf(`%%"%s"%%`, tag))
		}
	}

	return query
}

func applySorting(query *gorm.DB, params SearchParams) *gorm.DB {
	orderBy := strings.ToLower(params.OrderBy)
	if orderBy != "asc" && orderBy != "desc" {
		orderBy = "desc"
	}

	dbDialect := database.DB.Dialector.Name()

	switch params.SortBy {
	case "created_at":
		query = query.Order("created_at " + orderBy)
	case "updated_at":
		query = query.Order("updated_at " + orderBy)
	case "published_at":
		query = query.Order("published_at " + orderBy)
	case "title":
		if dbDialect == "postgres" {
			query = query.Order("data->>'title' " + orderBy)
		} else {
			query = query.Order("json_extract(data, '$.title') " + orderBy)
		}
	case "price":
		// Handle sorting by numeric fields like price
		if dbDialect == "postgres" {
			query = query.Order("(data->>'price')::numeric " + orderBy)
		} else {
			query = query.Order("CAST(json_extract(data, '$.price') AS REAL) " + orderBy)
		}
	default:
		query = query.Order("created_at " + orderBy)
	}

	return query
}

func GetSearchFacets(params SearchParams) (*SearchFacets, error) {
	facets := &SearchFacets{
		ContentTypes: make(map[string]int64),
		Statuses:     make(map[string]int64),
		DateRange:    &DateRangeFacet{},
	}

	query := database.DB.Model(&models.ContentEntry{})

	if params.Query != "" {
		_ = applyFullTextSearch(query, params)
	}

	var contentTypeCounts []struct {
		ContentTypeID uint
		Count         int64
	}
	database.DB.Model(&models.ContentEntry{}).
		Select("content_type_id, count(*) as count").
		Group("content_type_id").
		Scan(&contentTypeCounts)

	for _, ct := range contentTypeCounts {
		var contentType models.ContentType
		if err := database.DB.First(&contentType, ct.ContentTypeID).Error; err == nil {
			facets.ContentTypes[contentType.Name] = ct.Count
		}
	}

	var statusCounts []struct {
		Status string
		Count  int64
	}
	database.DB.Model(&models.ContentEntry{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusCounts)

	for _, sc := range statusCounts {
		facets.Statuses[sc.Status] = sc.Count
	}

	database.DB.Model(&models.ContentEntry{}).
		Select("MIN(created_at) as oldest, MAX(created_at) as newest").
		Scan(facets.DateRange)

	return facets, nil
}

func AdvancedFilter(filters map[string]any, params SearchParams) (*SearchResult, error) {
	query := database.DB.Model(&models.ContentEntry{})

	if len(params.ContentTypeIDs) > 0 {
		query = query.Where("content_type_id IN ?", params.ContentTypeIDs)
	}

	dbDialect := database.DB.Dialector.Name()

	for fieldName, value := range filters {
		switch v := value.(type) {
		case string:
			if dbDialect == "postgres" {
				query = query.Where("data->? = ?", fieldName, fmt.Sprintf(`"%s"`, v))
			} else {
				// SQLite
				query = query.Where("json_extract(data, ?) = ?", "$."+fieldName, v)
			}

		case []string:
			var conditions []string
			var args []interface{}
			for _, str := range v {
				if dbDialect == "postgres" {
					conditions = append(conditions, "data->? = ?")
					args = append(args, fieldName, fmt.Sprintf(`"%s"`, str))
				} else {
					conditions = append(conditions, "json_extract(data, ?) = ?")
					args = append(args, "$."+fieldName, str)
				}
			}
			if len(conditions) > 0 {
				whereClause := strings.Join(conditions, " OR ")
				query = query.Where(whereClause, args...)
			}

		case map[string]any:
			// Handle numeric range filters
			if dbDialect == "postgres" {
				if min, ok := v["min"]; ok {
					query = query.Where("(data->?)::numeric >= ?", fieldName, min)
				}
				if max, ok := v["max"]; ok {
					query = query.Where("(data->?)::numeric <= ?", fieldName, max)
				}
			} else {
				// SQLite - use CAST for numeric comparison
				if min, ok := v["min"]; ok {
					query = query.Where("CAST(json_extract(data, ?) AS REAL) >= ?", "$."+fieldName, min)
				}
				if max, ok := v["max"]; ok {
					query = query.Where("CAST(json_extract(data, ?) AS REAL) <= ?", "$."+fieldName, max)
				}
			}
		}
	}

	query = applySorting(query, params)

	var total int64
	query.Count(&total)

	offset := (params.Page - 1) * params.Limit
	query = query.Offset(offset).Limit(params.Limit)

	var entries []models.ContentEntry
	if err := query.Preload("Creator").Preload("Updater").Find(&entries).Error; err != nil {
		return nil, err
	}

	totalPages := total / int64(params.Limit)
	if total%int64(params.Limit) > 0 {
		totalPages++
	}

	return &SearchResult{
		Entries:    entries,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

func AutoComplete(field, prefix string, contentTypeID uint, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	var suggestions []string
	dbDialect := database.DB.Dialector.Name()

	query := database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ?", contentTypeID)

	if dbDialect == "postgres" {
		query.Distinct().
			Select(fmt.Sprintf("data->>'%s' as suggestion", field)).
			Where(fmt.Sprintf("data->>'%s' ILIKE ?", field), prefix+"%").
			Limit(limit).
			Pluck("suggestion", &suggestions)
	} else {
		query.Distinct().
			Select(fmt.Sprintf("json_extract(data, '$.%s') as suggestion", field)).
			Where(fmt.Sprintf("json_extract(data, '$.%s') LIKE ?", field), prefix+"%").
			Limit(limit).
			Pluck("suggestion", &suggestions)
	}

	return suggestions, nil
}

func SearchByRelation(entryID uint, relationType string) ([]models.ContentEntry, error) {
	var relations []models.ContentRelation
	if err := database.DB.Where("from_content_id = ? AND relation_type = ?", entryID, relationType).
		Find(&relations).Error; err != nil {
		return nil, err
	}

	var toIDs []uint
	for _, rel := range relations {
		toIDs = append(toIDs, rel.ToContentID)
	}

	if len(toIDs) == 0 {
		return []models.ContentEntry{}, nil
	}

	var entries []models.ContentEntry
	if err := database.DB.Where("id IN ?", toIDs).
		Preload("Creator").
		Preload("Updater").
		Find(&entries).Error; err != nil {
		return nil, err
	}

	return entries, nil
}

func GetPopularSearchTerms(limit int) ([]string, error) {
	return []string{}, nil
}

func HighlightMatches(text, query string) string {
	terms := strings.Fields(query)
	highlighted := text

	for _, term := range terms {
		highlighted = strings.ReplaceAll(highlighted, term, fmt.Sprintf("<mark>%s</mark>", term))
	}

	return highlighted
}

func ExtractSearchableText(entry models.ContentEntry) (string, error) {
	var data map[string]any
	if err := json.Unmarshal([]byte(entry.Data), &data); err != nil {
		return "", err
	}

	var texts []string
	for _, value := range data {
		if str, ok := value.(string); ok {
			texts = append(texts, str)
		}
	}

	return strings.Join(texts, " "), nil
}
