package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var categories map[string][]string

func main() {
	// Load categories from JSON file
	file, err := os.Open("data/scifi.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&categories); err != nil {
		panic(err)
	}

	// Initialize Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Serve static files
	e.Static("/", "static")

	// Define API routes
	e.GET("/api/categories", getCategories)
	e.GET("/api/random", getRandomElements)

	// Start the server
	e.Logger.Fatal(e.Start(":8080"))
}

func getCategories(c echo.Context) error {
	var categoryList []string
	for category := range categories {
		categoryList = append(categoryList, category)
	}

	// Generate HTML options for the select dropdown
	htmlOptions := ""
	for _, category := range categoryList {
		htmlOptions += `<option value="` + category + `">` + category + `</option>`
	}
	return c.HTML(http.StatusOK, htmlOptions)
}

func getRandomElements(c echo.Context) error {
	category := c.QueryParam("category")
	countParam := c.QueryParam("count")
	count, err := strconv.Atoi(countParam)
	if err != nil || count < 1 {
		count = 1
	}

	elements, ok := categories[category]
	if !ok {
		return c.HTML(http.StatusBadRequest, "Invalid category")
	}

	if count > len(elements) {
		count = len(elements)
	}

	randomElements := getRandomItems(elements, count)

	// Generate HTML list items for the results
	htmlResults := ""
	for _, element := range randomElements {
		htmlResults += `<li>` + element + `</li>`
	}
	return c.HTML(http.StatusOK, htmlResults)
}

func getRandomItems(list []string, count int) []string {
	perm := rand.Perm(len(list))
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = list[perm[i]]
	}
	return result
}
