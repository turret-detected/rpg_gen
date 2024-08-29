package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/samber/lo"
	"github.com/turret-detected/rpg-gen/app/api"
	"github.com/turret-detected/rpg-gen/app/gen"
	"gopkg.in/yaml.v3"
)

//go:embed static/*
var staticFolder embed.FS

// main
func main() {
	dataSource := flag.String("data", "data/demo.yaml", "data file to use")
	flag.Parse()

	// Load categories from YAML file
	data := gen.DataFileV1{}
	if strings.HasPrefix(*dataSource, "http") {
		req := lo.Must(http.NewRequest(http.MethodGet, *dataSource, nil))
		resp := lo.Must(http.DefaultClient.Do(req))
		if resp.StatusCode != http.StatusOK {
			fmt.Println(string(lo.Must(io.ReadAll(resp.Body))))
			panic(resp.Status)
		}
		lo.Must0(yaml.NewDecoder(resp.Body).Decode(&data))
	} else {
		file := lo.Must(os.Open(*dataSource))
		defer file.Close()
		lo.Must0(yaml.NewDecoder(file).Decode(&data))
	}

	// Initialize Echo server
	e := api.NewServer(gen.CreateGenerators(data), staticFolder)
	e.Logger.Fatal(e.Start(":8080"))
}
