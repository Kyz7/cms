package search_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/testutils"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestSearchEntriesHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Article", Slug: "article"}
	database.DB.Create(ct)

	entries := []struct {
		title   string
		content string
		tags    []string
		status  models.WorkflowStatus
	}{
		{"Introduction to Go Programming", "Go is a powerful language", []string{"go", "programming"}, models.StatusPublished},
		{"Advanced Go Techniques", "Learn advanced Go patterns", []string{"go", "advanced"}, models.StatusPublished},
		{"Python for Beginners", "Python is easy to learn", []string{"python", "beginner"}, models.StatusDraft},
		{"JavaScript Fundamentals", "Master JavaScript basics", []string{"javascript", "web"}, models.StatusPublished},
		{"Database Design Patterns", "SQL and NoSQL databases", []string{"database", "sql"}, models.StatusInReview},
	}

	for _, e := range entries {
		data := map[string]interface{}{
			"title":   e.title,
			"content": e.content,
			"tags":    e.tags,
		}
		jsonData, _ := json.Marshal(data)
		database.DB.Create(&models.ContentEntry{
			ContentTypeID: ct.ID,
			Data:          datatypes.JSON(jsonData),
			Status:        e.status,
			CreatedBy:     editor.ID,
		})
	}

	t.Run("Success - Basic search", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/entries?q=Go", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(data), 2)
	})

	t.Run("Success - Search with status filter", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/entries?q=&status=published", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(data), 3)
	})

	t.Run("Success - Search specific fields", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/entries?q=JavaScript&fields=title", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Search with pagination", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/entries?q=&page=1&limit=2", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.Equal(t, 1, result.Meta.Page)
		assert.Equal(t, 2, result.Meta.Limit)
	})

	t.Run("Success - Search with sorting", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/entries?q=&sort_by=created_at&order_by=asc", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Search with date range", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/entries?q=&from=2024-01-01&to=2024-12-31", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - No search parameters", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/entries", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)

		testutils.AssertError(t, resp, "BAD_REQUEST")
	})
}

func TestAdvancedSearchHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Product", Slug: "product"}
	database.DB.Create(ct)

	products := []struct {
		name     string
		price    float64
		category string
		stock    int
	}{
		{"Laptop", 1200.00, "Electronics", 10},
		{"Phone", 800.00, "Electronics", 25},
		{"Desk", 350.00, "Furniture", 5},
		{"Chair", 150.00, "Furniture", 15},
		{"Monitor", 400.00, "Electronics", 8},
	}

	for _, p := range products {
		data := map[string]interface{}{
			"name":     p.name,
			"price":    p.price,
			"category": p.category,
			"stock":    p.stock,
		}
		jsonData, _ := json.Marshal(data)
		database.DB.Create(&models.ContentEntry{
			ContentTypeID: ct.ID,
			Data:          datatypes.JSON(jsonData),
			Status:        models.StatusPublished,
			CreatedBy:     editor.ID,
		})
	}

	t.Run("Success - Advanced search with filters", func(t *testing.T) {
		body := map[string]interface{}{
			"query":  "",
			"status": "published",
			"filters": map[string]interface{}{
				"category": "Electronics",
			},
			"page":  1,
			"limit": 10,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/advanced", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(data), 3)
	})

	t.Run("Success - Price range filter", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "",
			"filters": map[string]interface{}{
				"price": map[string]interface{}{
					"min": 200,
					"max": 500,
				},
			},
			"sort_by":  "price",
			"order_by": "asc",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/advanced", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Multiple filters", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "",
			"filters": map[string]interface{}{
				"category": "Furniture",
				"price": map[string]interface{}{
					"max": 200,
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/advanced", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		data := result.Data.([]interface{})
		assert.Equal(t, 1, len(data))
	})

	t.Run("Success - Search with sorting by price", func(t *testing.T) {
		body := map[string]interface{}{
			"query":    "",
			"status":   "published",
			"sort_by":  "price",
			"order_by": "desc",
			"limit":    3,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/advanced", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})
}

func TestGetSearchFacetsHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	blog := &models.ContentType{Name: "Blog", Slug: "blog"}
	product := &models.ContentType{Name: "Product", Slug: "product"}
	database.DB.Create(blog)
	database.DB.Create(product)

	statuses := []models.WorkflowStatus{
		models.StatusDraft,
		models.StatusPublished,
		models.StatusInReview,
	}

	for i, status := range statuses {
		data := map[string]interface{}{
			"title": fmt.Sprintf("Entry %d", i),
		}
		jsonData, _ := json.Marshal(data)
		database.DB.Create(&models.ContentEntry{
			ContentTypeID: blog.ID,
			Data:          datatypes.JSON(jsonData),
			Status:        status,
			CreatedBy:     editor.ID,
		})
	}

	t.Run("Success - Get facets", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/facets", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		facets := result.Data.(map[string]interface{})
		assert.NotNil(t, facets["content_types"])
		assert.NotNil(t, facets["statuses"])
		assert.NotNil(t, facets["date_range"])
	})

	t.Run("Success - Get facets with query", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/facets?q=Entry", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})
}

func TestAutoCompleteHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Article", Slug: "article"}
	database.DB.Create(ct)

	titles := []string{
		"Introduction to Programming",
		"Introduction to Design",
		"Introduction to Marketing",
		"Advanced Programming",
		"Intermediate Design",
	}

	for _, title := range titles {
		data := map[string]interface{}{
			"title": title,
		}
		jsonData, _ := json.Marshal(data)
		database.DB.Create(&models.ContentEntry{
			ContentTypeID: ct.ID,
			Data:          datatypes.JSON(jsonData),
			Status:        models.StatusPublished,
			CreatedBy:     editor.ID,
		})
	}

	t.Run("Success - Autocomplete suggestions", func(t *testing.T) {
		url := fmt.Sprintf("/search/autocomplete?field=title&prefix=Intro&content_type_id=%d", ct.ID)
		resp, err := testutils.MakeRequest(app, "GET", url, nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.(map[string]interface{})
		suggestions := data["suggestions"].([]interface{})
		assert.Equal(t, 3, len(suggestions))
	})

	t.Run("Error - Missing required parameters", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/autocomplete?prefix=Test", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Success - No suggestions found", func(t *testing.T) {
		url := fmt.Sprintf("/search/autocomplete?field=title&prefix=Xyz&content_type_id=%d", ct.ID)
		resp, err := testutils.MakeRequest(app, "GET", url, nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		data := result.Data.(map[string]interface{})
		suggestions := data["suggestions"].([]interface{})
		assert.Equal(t, 0, len(suggestions))
	})
}

func TestSearchByRelationHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	blog := &models.ContentType{Name: "Blog", Slug: "blog"}
	category := &models.ContentType{Name: "Category", Slug: "category"}
	database.DB.Create(blog)
	database.DB.Create(category)

	blogData := map[string]interface{}{"title": "My Blog Post"}
	blogJSON, _ := json.Marshal(blogData)
	blogEntry := &models.ContentEntry{
		ContentTypeID: blog.ID,
		Data:          datatypes.JSON(blogJSON),
		Status:        models.StatusPublished,
		CreatedBy:     editor.ID,
	}
	database.DB.Create(blogEntry)

	for i := 1; i <= 3; i++ {
		catData := map[string]interface{}{"name": fmt.Sprintf("Category %d", i)}
		catJSON, _ := json.Marshal(catData)
		catEntry := &models.ContentEntry{
			ContentTypeID: category.ID,
			Data:          datatypes.JSON(catJSON),
			Status:        models.StatusPublished,
			CreatedBy:     editor.ID,
		}
		database.DB.Create(catEntry)

		database.DB.Create(&models.ContentRelation{
			FromContentID: blogEntry.ID,
			ToContentID:   catEntry.ID,
			RelationType:  "has_many",
		})
	}

	t.Run("Success - Search by relation", func(t *testing.T) {
		url := fmt.Sprintf("/search/entries/%d/related?type=has_many", blogEntry.ID)
		resp, err := testutils.MakeRequest(app, "GET", url, nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.([]interface{})
		assert.Equal(t, 3, len(data))
	})

	t.Run("Error - Missing relation type", func(t *testing.T) {
		url := fmt.Sprintf("/search/entries/%d/related", blogEntry.ID)
		resp, err := testutils.MakeRequest(app, "GET", url, nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Success - No relations found", func(t *testing.T) {
		url := fmt.Sprintf("/search/entries/%d/related?type=belongs_to", blogEntry.ID)
		resp, err := testutils.MakeRequest(app, "GET", url, nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		data := result.Data.([]interface{})
		assert.Equal(t, 0, len(data))
	})
}

func TestBulkSearchHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	blog := &models.ContentType{Name: "Blog", Slug: "blog"}
	article := &models.ContentType{Name: "Article", Slug: "article"}
	database.DB.Create(blog)
	database.DB.Create(article)

	for _, ct := range []*models.ContentType{blog, article} {
		for i := 1; i <= 2; i++ {
			data := map[string]interface{}{
				"title":   fmt.Sprintf("JavaScript Tutorial %d", i),
				"content": "Learn JavaScript programming",
			}
			jsonData, _ := json.Marshal(data)
			database.DB.Create(&models.ContentEntry{
				ContentTypeID: ct.ID,
				Data:          datatypes.JSON(jsonData),
				Status:        models.StatusPublished,
				CreatedBy:     editor.ID,
			})
		}
	}

	t.Run("Success - Bulk search across types", func(t *testing.T) {
		body := map[string]interface{}{
			"query":            "JavaScript",
			"content_type_ids": []uint{blog.ID, article.ID},
			"limit":            5,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/bulk", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.([]interface{})
		assert.Equal(t, 4, len(data))
	})

	t.Run("Success - Bulk search grouped by type", func(t *testing.T) {
		body := map[string]interface{}{
			"query":            "JavaScript",
			"content_type_ids": []uint{blog.ID, article.ID},
			"limit":            5,
			"group_by_type":    true,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/bulk", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.(map[string]interface{})
		results := data["results"].(map[string]interface{})
		assert.NotNil(t, results)
	})

	t.Run("Error - Missing query", func(t *testing.T) {
		body := map[string]interface{}{
			"content_type_ids": []uint{blog.ID},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/bulk", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})
}

func TestComplexSearchScenarios(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Product", Slug: "product"}
	database.DB.Create(ct)

	products := []map[string]interface{}{
		{
			"name":        "Gaming Laptop",
			"price":       1500,
			"brand":       "TechBrand",
			"category":    "Electronics",
			"tags":        []string{"gaming", "laptop", "computer"},
			"in_stock":    true,
			"rating":      4.5,
			"description": "High-performance gaming laptop with RTX graphics",
		},
		{
			"name":        "Office Laptop",
			"price":       800,
			"brand":       "OfficePro",
			"category":    "Electronics",
			"tags":        []string{"laptop", "office", "business"},
			"in_stock":    true,
			"rating":      4.0,
			"description": "Reliable laptop for office work",
		},
		{
			"name":        "Gaming Mouse",
			"price":       50,
			"brand":       "TechBrand",
			"category":    "Accessories",
			"tags":        []string{"gaming", "mouse", "rgb"},
			"in_stock":    false,
			"rating":      4.8,
			"description": "RGB gaming mouse with programmable buttons",
		},
	}

	for _, p := range products {
		jsonData, _ := json.Marshal(p)
		database.DB.Create(&models.ContentEntry{
			ContentTypeID: ct.ID,
			Data:          datatypes.JSON(jsonData),
			Status:        models.StatusPublished,
			CreatedBy:     editor.ID,
		})
	}

	t.Run("Complex - Search + Filter + Sort", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "gaming",
			"filters": map[string]interface{}{
				"in_stock": true,
			},
			"sort_by":  "price",
			"order_by": "desc",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/advanced", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		data := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(data), 1)
	})

	t.Run("Complex - Multiple tags filter", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/search/entries?q=&tags=gaming,laptop", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Complex - Brand and category filter", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "",
			"filters": map[string]interface{}{
				"brand":    "TechBrand",
				"category": "Electronics",
			},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/search/advanced", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		data := result.Data.([]interface{})
		assert.Equal(t, 1, len(data))
	})
}
