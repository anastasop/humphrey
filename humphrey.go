package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/PuerkitoBio/goquery"
)

// humphrey -tmpl "{{.key|println}}{{range .img}}{{.|println}}{{end}}" -page http://www.oldpicsarchive.com/10-colorized-photos-a
// udrey-hepburn "img:.pagination a:href" | humphrey -tmpl "{{.img|println}}" "img:.post-single-content img:src"

// rule represents a parsing rule for an html page
// Selector is a css selector. The parser applies the selector
// to the html and extracts the text of the elements matched
// or the text of the named Attributes if present.
// Name is the key of the result for the generated result map.
type rule struct {
	Name      string
	Selector  string
	Attribute string
}

// newRule builds a new rule from text. The three parts
// should be separated by a colon
func newRule(s string) (*rule, error) {
	toks := strings.SplitN(s, ":", 3)
	switch len(toks) {
	case 2:
		return &rule{toks[0], toks[1], ""}, nil
	case 3:
		return &rule{toks[0], toks[1], toks[2]}, nil
	}
	return nil, fmt.Errorf("can't parse rule: %s", s)
}

// apply the rule to the document and write the results to map
// the result is stored according to options arrays. If true
// it is always an array, maybe empty or with a single element.
// Otherwise
// if the rule selector matches only one element the result is a string
// if it matches many elements, the result is an array.
// if it matched nothing, the results is nil
func (r *rule) apply(doc *goquery.Document, m map[string]interface{}, as_array bool) {
	var vals []string

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

	if as_array || len(vals) > 1 {
		m[r.Name] = vals
	} else {
		if len(vals) == 0 {
			m[r.Name] = nil
		} else {
			m[r.Name] = vals[0]
		}
	}
}

// download uses the http to download the page of url u
// and returns the results as an io.Reader
// It returns a non-nil error if downloading fails
// or the http response code is not 200
func download(u string) (io.Reader, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got http %d instead of 200 for url: %s",
			resp.StatusCode, u)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), nil
}

// downloadAndApplyRules tries to download the url u and apply the rules
// it return error if download or parsing fails
func downloadAndApplyRules(u string, rules []*rule, as_array bool) (map[string]interface{}, error) {
	r, err := download(u)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	for _, rr := range rules {
		rr.apply(doc, m, as_array)
	}

	return m, nil
}

var key = flag.String("key", "key", "the name for the url in output map")
var tmpl = flag.String("tmpl", "", "a text/template for output instead of json")
var page = flag.String("page", "", "the url to scrap. If not set it reads all lines from stdin")
var pretty = flag.Bool("pretty", false, "pretty print json")
var strict = flag.Bool("strict", true, "If a urls fails then stop the program")
var arrays = flag.Bool("arrays", false, "Always store the result as array. Mostly useful with templates")

func usage() {
	fmt.Fprintf(os.Stderr, "usage: humphrey [options] [rules]\n")
	fmt.Fprintf(os.Stderr, "rules:\n")
	fmt.Fprintf(os.Stderr, "  key:selector[:attribute]\n")
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.SetPrefix("humphrey: ")
	log.SetFlags(0)
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}

	var rules []*rule
	for _, s := range flag.Args() {
		if r, err := newRule(s); err == nil {
			rules = append(rules, r)
		} else {
			log.Fatal(err)
		}
	}

	var t *template.Template
	var enc *json.Encoder
	if *tmpl != "" {
		tt, err := template.New("output").Parse(*tmpl)
		if err != nil {
			log.Fatal(err)
		}
		t = tt
	} else {
		enc = json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		if *pretty {
			enc.SetIndent("", "  ")
		}
	}

	var m map[string]interface{}
	var err error

	var scanner *bufio.Scanner
	if *page != "" {
		scanner = bufio.NewScanner(bytes.NewBufferString(*page))
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}
	for scanner.Scan() {
		u := strings.TrimSpace(scanner.Text())
		m, err = downloadAndApplyRules(u, rules, *arrays)
		if err == nil {
			m[*key] = u
			if t != nil {
				if err := t.Execute(os.Stdout, m); err != nil {
					log.Fatal(err)
				}
			} else if enc != nil {
				if err := enc.Encode(m); err != nil {
					log.Fatal(err)
				}
			}
		} else {
			if *strict {
				log.Fatal(err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("reading standard input:", err)
	}
}
