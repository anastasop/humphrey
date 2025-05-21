package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// rule represents a parsing rule for an html page.
// Selector is a css selector. The parser applies the selector
// to the html and extracts the text of the elements matched
// or the text of the named Attribute if present.
// Name is the key of the result for the generated result map.
type rule struct {
	Name      string
	Selector  string
	Attribute string
}

// newRule builds a new rule from a string containing the three parts
// separated by a slash for example links/a or links/a/href .
func newRule(s string) (*rule, error) {
	toks := strings.Split(s, "/")
	if len(toks) != 2 && len(toks) != 3 {
		return nil, fmt.Errorf("can't parse rule: %s", s)
	}
	if len(toks) == 2 {
		toks = append(toks, "")
	}
	for i := 0; i < len(toks); i++ {
		toks[i] = strings.TrimSpace(toks[i])
	}
	if toks[0] == "" || toks[1] == "" {
		return nil, fmt.Errorf("rule %s has empty parts", s)
	}

	return &rule{toks[0], toks[1], toks[2]}, nil
}

// apply aplpies the rule to the document and write the resulting array to map.
func (r *rule) apply(m map[string]any, doc *goquery.Document) {
	vals := make([]string, 0)

	doc.Find(r.Selector).Each(func(i int, s *goquery.Selection) {
		var val string
		if r.Attribute == "" {
			val = s.Text()
		} else {
			if v, exists := s.Attr(r.Attribute); exists {
				val = v
			}
		}
		vals = append(vals, html.UnescapeString(strings.TrimSpace(val)))
	})

	addValues(m, r.Name, vals)
}

// download fetches the page of u and returns it as an io.Reader.
// Expects to get an HTTP 200.
func download(u string) (io.Reader, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got http %d instead of 200 for url: %s",
			resp.StatusCode, u)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), nil
}

// applyRules tries to download the url u and apply the rules.
func applyRules(m map[string]any, u string, rules []*rule) error {
	r, err := download(u)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return err
	}

	for _, rr := range rules {
		rr.apply(m, doc)
	}

	return nil
}

var key = flag.String("key", "key", "the name for the url in output map")
var rawOutput = flag.Bool("r", false, "text output instead of json")
var ruleNames []string

func usage() {
	fmt.Fprintf(os.Stderr, "usage: humphrey [options] rules... url\n")
	fmt.Fprintf(os.Stderr, "rules:\n")
	fmt.Fprintf(os.Stderr, "  key/selector[/attribute]\n")
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.SetPrefix("humphrey: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	// we need at least one rule and the url
	if flag.NArg() < 2 {
		usage()
	}

	var rules []*rule
	for i := 0; i < flag.NArg()-1; i++ {
		if r, err := newRule(flag.Arg(i)); err == nil {
			rules = append(rules, r)
			ruleNames = append(ruleNames, r.Name)
		} else {
			log.Fatal(err)
		}
	}
	page := flag.Arg(flag.NArg() - 1)

	m := make(map[string]any)
	if err := prepare(m, ruleNames); err != nil {
		log.Fatal(err)
	}
	if err := applyRules(m, page, rules); err != nil {
		log.Fatal(err)
	}
	m[*key] = page

	if *rawOutput {
		printRecursively(m)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(m); err != nil {
			log.Fatal(err)
		}
	}
}

// prepare initializes the map values to accept nested names.
// Examples:
// links/a/href
//
//	{
//	  "links": [""]
//	}
//
// links.href/a/href links.text/a
//
//	{
//	  "links": [
//	    {
//	      "href": "",
//	      "text": ""
//	    }
//	  ]
//	}
//
// page.links.href/a/href page.links.text/a
//
//	{
//	  "page": {
//	    "links": [
//	      {
//	        "href": "",
//	        "text": ""
//	      }
//	    ]
//	  }
func prepare(m map[string]any, names []string) error {
	for _, name := range names {
		var parent map[string]any = m
		var terminals []string
		parts := strings.Split(name, ".")
		if len(parts) > 2 {
			for i, s := range parts[0 : len(parts)-2] {
				if p, ok := parent[s]; ok {
					if pp, ok := p.(map[string]any); ok {
						parent = pp
					} else {
						return fmt.Errorf("name mismatch: %s is not map[string]any: rules: %s", strings.Join(parts[0:i], "."), ruleNames)
					}
				} else {
					pm := make(map[string]any)
					parent[s] = pm
					parent = pm
				}
			}
			terminals = parts[len(parts)-2:]
		} else {
			terminals = parts
		}

		if len(terminals) == 1 {
			if currValsAny, ok := parent[terminals[0]]; ok {
				if _, ok := currValsAny.([]string); !ok {
					return fmt.Errorf("name mismatch: %s is not []string: rules: %s", strings.Join(parts, "."), ruleNames)
				}
			} else {
				parent[terminals[0]] = make([]string, 0)
			}
		} else {
			if currValsAny, ok := parent[terminals[0]]; ok {
				if _, ok := currValsAny.([]map[string]string); !ok {
					return fmt.Errorf("name mismatch: %s is not []map[string]string: rules: %s", strings.Join(parts[0:len(parts)-1], "."), ruleNames)
				}
			} else {
				parent[terminals[0]] = make([]map[string]string, 0)
			}
		}
	}

	return nil
}

// addValues adds to map the values under name. Assumes map is prepared.
func addValues(m map[string]any, name string, vals []string) {
	parts := strings.Split(name, ".")
	var terminals []string
	var parent map[string]any = m
	if len(parts) > 2 {
		for _, s := range parts[0 : len(parts)-2] {
			parent = parent[s].(map[string]any)
		}
		terminals = parts[len(parts)-2:]
	} else {
		terminals = parts
	}

	if len(terminals) == 1 {
		currVals := parent[terminals[0]].([]string)
		for i, v := range vals {
			if i < len(currVals) {
				currVals[i] = v
			} else {
				currVals = append(currVals, v)
			}
		}
		parent[terminals[0]] = currVals
	} else {
		objs := parent[terminals[0]].([]map[string]string)
		for i, v := range vals {
			if i < len(objs) {
				objs[i][terminals[1]] = v
			} else {
				mm := make(map[string]string)
				mm[terminals[1]] = v
				objs = append(objs, mm)
			}
		}
		parent[terminals[0]] = objs
	}
}

// printRecursively descends into m and prints all string values.
// m must be prepared.
func printRecursively(m map[string]any) {
	for k, t := range m {
		switch v := t.(type) {
		case string:
			fmt.Println(v)
		case []string:
			for _, s := range v {
				fmt.Println(s)
			}
		case []map[string]string:
			for _, o := range v {
				for _, ov := range o {
					fmt.Println(ov)
				}
			}
		case map[string]any:
			printRecursively(v)
		default:
			log.Fatalf("key %s has map type %T", k, v)
		}
	}
}
