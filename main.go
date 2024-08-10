package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	wrand "github.com/mroth/weightedrand/v2"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

type DataFileV1 struct {
	Version    int            `yaml:"version"`
	Generators []*GeneratorV1 `yaml:"generators"`
}

type GeneratorV1 struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Entries []any  `yaml:"entries"`
	chooser *wrand.Chooser[string, int]
}

type GeneratorType string

const (
	Weighted   GeneratorType = "weighted"
	Unweighted GeneratorType = "unweighted"
)

var (
	DataTable DataFileV1 = DataFileV1{}
)

func createGenerators(data DataFileV1) DataFileV1 {
	for _, generator := range data.Generators {
		choices := make([]wrand.Choice[string, int], len(generator.Entries))
		generatorType := GeneratorType(generator.Type)
		switch generatorType {
		case Weighted:
			for i, choice := range generator.Entries {
				choiceList := choice.([]any)
				entry := choiceList[0].(string)
				weight := choiceList[1].(float64)
				choices[i] = wrand.NewChoice(entry, int(weight*100))
			}
		case Unweighted:
			for i, choice := range generator.Entries {
				entry := choice.(string)
				choices[i] = wrand.NewChoice(entry, 1)
			}
		default:
			choices = []wrand.Choice[string, int]{{
				Item:   fmt.Sprint("invalid type: ", generator.Type, ". THIS IS A BUG"),
				Weight: 1,
			}}
		}
		generator.chooser = lo.Must(wrand.NewChooser(choices...))
	}
	return data
}

func main() {
	filePath := flag.String("data", "data/demo.yaml", "data file to use")
	flag.Parse()

	// Load categories from YAML file
	file := lo.Must(os.Open(*filePath))
	defer file.Close()
	data := DataFileV1{}
	lo.Must0(yaml.NewDecoder(file).Decode(&data))
	DataTable = createGenerators(data)

	// Initialize Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("data", &DataTable)
			return next(c)
		}
	})

	// Serve static files
	e.Static("/", "static")

	// Define API routes
	e.GET("/api/categories", getCategories)
	e.GET("/api/random", getRandomElements)

	admin := e.Group("/admin", middleware.KeyAuth(func(auth string, c echo.Context) (bool, error) {
		return auth == os.Getenv("RPG_ADMIN_KEY"), nil
	}))
	admin.PUT("/upload", putUpload)

	// Start the server
	e.Logger.Fatal(e.Start(":8080"))
}

func getCategories(c echo.Context) error {
	data := c.Get("data").(*DataFileV1)

	var categoryList []string
	for _, generator := range data.Generators {
		categoryList = append(categoryList, generator.Name)
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

	data := c.Get("data").(*DataFileV1)

	coll, ok := lo.Find(data.Generators, func(generator *GeneratorV1) bool {
		return generator.Name == category
	})
	if !ok {
		return c.HTML(http.StatusBadRequest, "Invalid category")
	}

	randomElements := make([]string, count)
	for i := range randomElements {
		randomElements[i] = coll.chooser.Pick()
	}

	// Generate HTML list items for the results
	htmlResults := ""
	for _, element := range randomElements {
		htmlResults += `<li>` + element + `</li>`
	}
	return c.HTML(http.StatusOK, htmlResults)
}

func putUpload(c echo.Context) error {
	newData := DataFileV1{}
	lo.Must0(yaml.NewDecoder(c.Request().Body).Decode(&newData))
	DataTable = createGenerators(newData)
	return c.String(http.StatusOK, "data updated")
}
