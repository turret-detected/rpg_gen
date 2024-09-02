package api

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/samber/lo"
	"github.com/turret-detected/rpg-gen/app/gen"
	"gopkg.in/yaml.v3"
)

const (
	RandomMin int = 1
	RandomMax int = 50
)

var (
	DataTable gen.DataFileV1 = gen.DataFileV1{}
)

// echo boilerplate
type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// GET /api/random
//
// Given a category and a count, generates HTML list of random elements
func getRandomElements(c echo.Context) error {
	category := c.QueryParam("category")
	countParam := c.QueryParam("count")
	count, err := strconv.Atoi(countParam)
	if err != nil || count < RandomMin {
		count = RandomMin
	}
	if count > RandomMax {
		count = RandomMax
	}

	data := c.Get("data").(*gen.DataFileV1)

	coll, ok := lo.Find(data.Generators, func(generator *gen.GeneratorV1) bool {
		return generator.Name == category
	})
	if !ok {
		return c.HTML(http.StatusBadRequest, `<p>Invalid category</p>`)
	}

	randomElements := make([]string, count)
	for i := range randomElements {
		randomElements[i] = coll.Chooser.Pick()
	}

	// Generate HTML list items for the results
	htmlResults := ""
	for _, element := range randomElements {
		htmlResults += `<li>` + element + `</li>`
	}
	return c.HTML(http.StatusOK, htmlResults)
}

// GET /api/categories
//
// Uses the data file to generate a list of HTML options containing all available categories
func getCategories(c echo.Context) error {
	data := c.Get("data").(*gen.DataFileV1)

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

// PUT /admin/upload
//
// Parses the uploaded yaml file and applies its contents to the server
func putUpload(c echo.Context) error {
	newData := gen.DataFileV1{}
	lo.Must0(yaml.NewDecoder(c.Request().Body).Decode(&newData))
	DataTable = gen.CreateGenerators(newData)
	return c.String(http.StatusOK, "data updated")
}

// GET /generator/:generatorName
//
// Loads the HTML template for the given generator
func getGenerator(c echo.Context) error {
	category := c.Param("generatorName")

	_, ok := lo.Find(DataTable.Generators, func(generator *gen.GeneratorV1) bool {
		return generator.Name == category
	})
	if !ok {
		return c.String(http.StatusBadRequest, "no generator by this name")
	}

	return c.Render(http.StatusOK, "gen.html", map[string]string{
		"name": category,
	})
}

func NewServer(data gen.DataFileV1, staticFiles embed.FS) *echo.Echo {
	DataTable = data

	t := &Template{
		templates: template.Must(template.ParseFS(staticFiles, "templates/*.html")),
	}

	e := echo.New()
	e.Renderer = t
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("data", &DataTable)
			return next(c)
		}
	})

	e.StaticFS("/", echo.MustSubFS(staticFiles, "static"))
	e.GET("/generator/:generatorName", getGenerator)
	e.GET("/api/categories", getCategories)
	e.GET("/api/random", getRandomElements)

	admin := e.Group("/admin", middleware.KeyAuth(func(auth string, c echo.Context) (bool, error) {
		return auth == os.Getenv("RPG_ADMIN_KEY"), nil
	}))
	admin.PUT("/upload", putUpload)
	return e
}
