package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/token"
	"log"
	"sort"
	"strings"

	"modernc.org/y"
)

var termName = map[string]string{
	"identifier":         "identifier",
	"singleAtIdentifier": "singleAtIdentifier",
	"doubleAtIdentifier": "doubleAtIdentifier",
	"invalid":            "invalid",
	"hintComment":        "hintComment",
	"stringLit":          "stringLit",
	"floatLit":           "floatLit",
	"decLit":             "decLit",
	"intLit":             "intLit",
	"hexLit":             "hexLit",
	"bitLit":             "bitLit",
}

func main() {
	yaccFile := flag.String("in", "parser.y", "yacc file")
	recursive := flag.Bool("recursive", false, "automatically add sub items")
	_include := flag.String("include", "", "list of symbols needs to be output, separated by comma")
	_exclude := flag.String("exclude", "Identifier,Expression,BitExpr", "list of symbols that should not be output, separated by comma")
	flag.Parse()
	include := strings.Split(*_include, ",")
	exclude := strings.Split(*_exclude, ",")
	sort.Strings(exclude)

	p, err := y.ProcessFile(token.NewFileSet(), *yaccFile, &y.Options{})
	if err != nil {
		log.Fatal(err)
	}

	lines := &bytes.Buffer{}
	dup := make(map[string]struct{}, len(include))
	for i := 0; i < len(include); i++ {
		v := p.Syms[include[i]]
		if v == nil || v.IsTerminal {
			continue
		}
		if k := sort.SearchStrings(exclude, v.Name); k != len(exclude) && exclude[k] == v.Name {
			continue
		}
		dup[v.Name] = struct{}{}

		lines.Reset()

		fmt.Fprintf(lines, "%s ::=\n", v.Name)
		for i, rule := range v.Rules {
			fmt.Fprintf(lines, "\t ")
			if i > 0 {
				fmt.Fprintf(lines, "| ")
			}
			if len(rule.Components) == 0 {
				fmt.Fprintf(lines, "/* empty */")
			}
			for _, c := range rule.Components {
				sym := p.Syms[c]
				if sym.IsTerminal {
					v := termName[sym.Name]
					if v != "" {
						fmt.Fprintf(lines, "%s ", v)
					} else if sym.LiteralString != "" {
						fmt.Fprintf(lines, "%s ", sym.LiteralString)
					} else {
						fmt.Fprintf(lines, "%s ", sym.Name)
					}
				} else {
					fmt.Fprintf(lines, "%s ", sym.Name)
				}
				if *recursive {
					if _, ok := dup[sym.Name]; !ok {
						include = append(include, sym.Name)
						dup[sym.Name] = struct{}{}
					}
				}
			}
			fmt.Fprintf(lines, "\n")
		}

		fmt.Println(lines.String())
	}
}
