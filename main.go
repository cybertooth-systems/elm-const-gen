package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/hashicorp/go-envparse"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const moduleTemplate = `{-
-----------
-- WARNING:
-- This is a generated file and any manual edits may be lost without warning.
-----------
-}
module {{ .Name }} exposing ({{ .ExposeList }})
{{ .FunBlocks }}`

const funTemplate = `


{{ .Name }} : String
{{ .Name }} =
    "{{ .Value }}"
`

type Module struct {
	Name, ExposeList, FunBlocks string
}

type FunBlock struct {
	Name, Value string
}

func main() {
	modT := template.Must(template.New("module").Parse(moduleTemplate))
	funT := template.Must(template.New("function").Parse(funTemplate))

	var (
		envFile = flag.String("e", ".env", "env file to parse")
		srcDir  = flag.String("s", "./src", "directory to write constants")
		modName = flag.String("n", "ConstGen", "name of the module to write")
	)
	flag.Parse()

	ef, err := os.Open(*envFile)
	if err != nil {
		fmt.Printf("cannot open env file: %v - %v\n", *envFile, err)
		os.Exit(1)
	}

	// parse env
	envs, err := envparse.Parse(ef)
	if err != nil {
		fmt.Printf("cannot parse env file: %v - %v\n", *envFile, err)
		os.Exit(1)
	}

	if len(envs) == 0 {
		fmt.Printf("no envs to parse from file: %v - exiting\n", *envFile)
		os.Exit(0)
	}

	// convert to elm-ish syntax
	elmMap := map[string]string{}
	elmKeys := []string{}

	for k, v := range envs {
		ks := strings.Split(strings.ToLower(k), "_")
		var newK string
		newK += ks[0]
		if len(ks) > 1 {
			for _, e := range ks[1:] {
				c := cases.Title(language.AmericanEnglish)
				newK += c.String(e)
			}
		}

		elmKeys = append(elmKeys, newK)
		elmMap[newK] = v
	}

	// render sorted elm functions
	sort.Strings(elmKeys)
	funBlocks := new(strings.Builder)

	for _, k := range elmKeys {
		v := elmMap[k]
		fb := FunBlock{Name: k, Value: v}

		err := funT.Execute(funBlocks, fb)
		if err != nil {
			fmt.Printf(
				"unrecoverable error processing env vars: %v - %v\n",
				*envFile,
				err,
			)
			os.Exit(1)
		}
	}

	// render entire module
	module := new(strings.Builder)
	m := Module{
		Name:       *modName,
		ExposeList: strings.Join(elmKeys, ", "),
		FunBlocks:  funBlocks.String(),
	}

	if err := modT.Execute(module, m); err != nil {
		fmt.Printf(
			"unrecoverable error processing env vars: %v - %v\n",
			*envFile,
			err,
		)
		os.Exit(1)
	}

	// write out rendered module to file
	outPath := filepath.Clean(fmt.Sprintf("%s/%s.elm", *srcDir, *modName))
	out, err := os.Create(outPath)
	if err != nil {
		fmt.Printf(
			"cannot open and create output file %v - %v\n",
			outPath,
			err,
		)
		os.Exit(1)
	}
	if _, err := out.Write([]byte(module.String())); err != nil {
		fmt.Printf(
			"cannot write output to file %v - %v\n",
			outPath,
			err,
		)
		os.Exit(1)
	}

	// results if we made it here...
	fmt.Printf(
		"%v vars from %v were exported to %v\n",
		len(elmKeys),
		*envFile,
		outPath,
	)
}
