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
	toks := strings.SplitN(s, "/", 3)
	switch len(toks) {
	case 2:
		return &rule{toks[0], toks[1], ""}, nil
	case 3:
		return &rule{toks[0], toks[1], toks[2]}, nil
	}
	return nil, fmt.Errorf("can't parse rule: %s", s)
}

// apply aplpies the rule to the document and write the resulting array to map.
func (r *rule) apply(doc *goquery.Document, m map[string]any) {
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

	m[r.Name] = vals
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
func applyRules(u string, rules []*rule) (map[string]any, error) {
	r, err := download(u)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	m := make(map[string]any)
	for _, rr := range rules {
		rr.apply(doc, m)
	}

	return m, nil
}

var key = flag.String("key", "key", "the name for the url in output map")
var rawOutput = flag.Bool("r", false, "text output instead of json")

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
		} else {
			log.Fatal(err)
		}
	}
	page := flag.Arg(flag.NArg() - 1)

	m, err := applyRules(page, rules)
	if err != nil {
		log.Fatal(err)
	}
	m[*key] = page

	if *rawOutput {
		for _, r := range rules {
			if strs, ok := m[r.Name].([]string); ok {
				for _, s := range strs {
					fmt.Println(s)
				}
			}
		}
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(m); err != nil {
			log.Fatal(err)
		}
	}
}
