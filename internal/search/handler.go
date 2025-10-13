package search

import (
	"strconv"
	"strings"

	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
)

func SearchEntriesHandler(c *fiber.Ctx) error {
	params := SearchParams{
		Query:    c.Query("q", ""),
		Status:   c.Query("status", ""),
		Page:     c.QueryInt("page", 1),
		Limit:    c.QueryInt("limit", 10),
		SortBy:   c.Query("sort_by", "created_at"),
		OrderBy:  c.Query("order_by", "desc"),
		FromDate: c.Query("from", ""),
		ToDate:   c.Query("to", ""),
	}

	if ctIDs := c.Query("content_type_ids"); ctIDs != "" {
		ids := strings.Split(ctIDs, ",")
		for _, id := range ids {
			if idUint, err := strconv.ParseUint(id, 10, 32); err == nil {
				params.ContentTypeIDs = append(params.ContentTypeIDs, uint(idUint))
			}
		}
	}

	if fields := c.Query("fields"); fields != "" {
		params.Fields = strings.Split(fields, ",")
	}

	if tags := c.Query("tags"); tags != "" {
		params.Tags = strings.Split(tags, ",")
	}

	if createdBy := c.QueryInt("created_by", 0); createdBy > 0 {
		params.CreatedBy = uint(createdBy)
	}

	if len(c.Context().QueryArgs().String()) == 0 {
		return response.BadRequest(c, "At least one search parameter is required", nil)
	}

	result, err := FullTextSearch(params)
	if err != nil {
		return response.InternalError(c, "Search failed: "+err.Error())
	}

	meta := &response.Meta{
		Page:       result.Page,
		Limit:      result.Limit,
		Total:      result.Total,
		TotalPages: result.TotalPages,
	}

	return response.SuccessWithMeta(c, result.Entries, meta, "Search completed successfully")
}

func AdvancedSearchHandler(c *fiber.Ctx) error {
	var body struct {
		Query          string                 `json:"query"`
		ContentTypeIDs []uint                 `json:"content_type_ids"`
		Fields         []string               `json:"fields"`
		Status         string                 `json:"status"`
		CreatedBy      uint                   `json:"created_by"`
		Tags           []string               `json:"tags"`
		FromDate       string                 `json:"from_date"`
		ToDate         string                 `json:"to_date"`
		Filters        map[string]interface{} `json:"filters"`
		Page           int                    `json:"page"`
		Limit          int                    `json:"limit"`
		SortBy         string                 `json:"sort_by"`
		OrderBy        string                 `json:"order_by"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Page <= 0 {
		body.Page = 1
	}
	if body.Limit <= 0 {
		body.Limit = 10
	}
	if body.SortBy == "" {
		body.SortBy = "created_at"
	}
	if body.OrderBy == "" {
		body.OrderBy = "desc"
	}

	params := SearchParams{
		Query:          body.Query,
		ContentTypeIDs: body.ContentTypeIDs,
		Fields:         body.Fields,
		Status:         body.Status,
		CreatedBy:      body.CreatedBy,
		Tags:           body.Tags,
		FromDate:       body.FromDate,
		ToDate:         body.ToDate,
		Page:           body.Page,
		Limit:          body.Limit,
		SortBy:         body.SortBy,
		OrderBy:        body.OrderBy,
	}

	var result *SearchResult
	var err error

	if len(body.Filters) > 0 {
		result, err = AdvancedFilter(body.Filters, params)
	} else {
		result, err = FullTextSearch(params)
	}

	if err != nil {
		return response.InternalError(c, "Search failed: "+err.Error())
	}

	meta := &response.Meta{
		Page:       result.Page,
		Limit:      result.Limit,
		Total:      result.Total,
		TotalPages: result.TotalPages,
	}

	return response.SuccessWithMeta(c, result.Entries, meta, "Search completed successfully")
}

func GetSearchFacetsHandler(c *fiber.Ctx) error {
	params := SearchParams{
		Query: c.Query("q", ""),
	}

	if ctIDs := c.Query("content_type_ids"); ctIDs != "" {
		ids := strings.Split(ctIDs, ",")
		for _, id := range ids {
			if idUint, err := strconv.ParseUint(id, 10, 32); err == nil {
				params.ContentTypeIDs = append(params.ContentTypeIDs, uint(idUint))
			}
		}
	}

	facets, err := GetSearchFacets(params)
	if err != nil {
		return response.InternalError(c, "Failed to get facets")
	}

	return response.Success(c, facets, "Facets retrieved successfully")
}

func AutoCompleteHandler(c *fiber.Ctx) error {
	field := c.Query("field")
	prefix := c.Query("prefix")
	contentTypeID := c.QueryInt("content_type_id", 0)
	limit := c.QueryInt("limit", 10)

	if field == "" || prefix == "" || contentTypeID == 0 {
		return response.ValidationError(c, map[string]string{
			"field":           "field is required",
			"prefix":          "prefix is required",
			"content_type_id": "content_type_id is required",
		})
	}

	suggestions, err := AutoComplete(field, prefix, uint(contentTypeID), limit)
	if err != nil {
		return response.InternalError(c, "Autocomplete failed")
	}

	return response.Success(c, fiber.Map{
		"suggestions": suggestions,
		"field":       field,
		"prefix":      prefix,
	}, "Autocomplete results")
}

func SearchByRelationHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	relationType := c.Query("type", "")
	if relationType == "" {
		return response.ValidationError(c, map[string]string{
			"type": "relation type is required",
		})
	}

	entries, err := SearchByRelation(uint(entryID), relationType)
	if err != nil {
		return response.InternalError(c, "Failed to search relations")
	}

	return response.Success(c, entries, "Related entries retrieved successfully")
}

func SearchSuggestionsHandler(c *fiber.Ctx) error {
	query := c.Query("q", "")
	if query == "" {
		return response.BadRequest(c, "Query parameter is required", nil)
	}

	suggestions := []string{
		query + " tutorial",
		query + " guide",
		query + " example",
	}

	return response.Success(c, fiber.Map{
		"suggestions": suggestions,
		"query":       query,
	}, "Search suggestions")
}

func BulkSearchHandler(c *fiber.Ctx) error {
	var body struct {
		Query          string `json:"query"`
		ContentTypeIDs []uint `json:"content_type_ids"`
		Limit          int    `json:"limit"`
		GroupByType    bool   `json:"group_by_type"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Query == "" {
		return response.ValidationError(c, map[string]string{
			"query": "query is required",
		})
	}

	if body.Limit <= 0 {
		body.Limit = 5
	}

	params := SearchParams{
		Query:          body.Query,
		ContentTypeIDs: body.ContentTypeIDs,
		Page:           1,
		Limit:          body.Limit * len(body.ContentTypeIDs),
	}

	result, err := FullTextSearch(params)
	if err != nil {
		return response.InternalError(c, "Bulk search failed")
	}

	if body.GroupByType {
		grouped := make(map[uint][]interface{})
		for _, entry := range result.Entries {
			grouped[entry.ContentTypeID] = append(grouped[entry.ContentTypeID], entry)
		}

		return response.Success(c, fiber.Map{
			"results": grouped,
			"total":   result.Total,
		}, "Bulk search completed")
	}

	return response.Success(c, result.Entries, "Bulk search completed")
}

func ExportSearchResultsHandler(c *fiber.Ctx) error {
	format := c.Query("format", "json")

	params := SearchParams{
		Query:   c.Query("q", ""),
		Status:  c.Query("status", ""),
		Page:    1,
		Limit:   1000,
		SortBy:  c.Query("sort_by", "created_at"),
		OrderBy: c.Query("order_by", "desc"),
	}

	result, err := FullTextSearch(params)
	if err != nil {
		return response.InternalError(c, "Export failed")
	}

	if format == "csv" {
		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", "attachment; filename=search-results.csv")

		return c.SendString("CSV export not yet implemented")
	}

	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename=search-results.json")

	return c.JSON(result.Entries)
}

func SearchStatsHandler(c *fiber.Ctx) error {
	stats := fiber.Map{
		"total_searches":         0,
		"unique_queries":         0,
		"avg_results_per_search": 0,
		"popular_queries":        []string{},
		"zero_result_queries":    []string{},
	}

	return response.Success(c, stats, "Search statistics")
}
