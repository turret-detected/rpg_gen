package gen

import (
	"fmt"

	wrand "github.com/mroth/weightedrand/v2"
	"github.com/samber/lo"
)

type DataFileV1 struct {
	Version    int            `yaml:"version"`
	Generators []*GeneratorV1 `yaml:"generators"`
}

type GeneratorV1 struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Entries []any  `yaml:"entries"`
	Chooser *wrand.Chooser[string, int]
}

type GeneratorType string

const (
	Weighted   GeneratorType = "weighted"
	Unweighted GeneratorType = "unweighted"
)

// Adds a weighted random chooser to DataFileV1 generators
func CreateGenerators(data DataFileV1) DataFileV1 {
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
		generator.Chooser = lo.Must(wrand.NewChooser(choices...))
	}
	return data
}
