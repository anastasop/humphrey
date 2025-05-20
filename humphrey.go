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
	"text/template"

	"github.com/PuerkitoBio/goquery"
)

// humphrey -tmpl "{{.key|println}}{{range .img}}{{.|println}}{{end}}" -page http://www.oldpicsarchive.com/10-colorized-photos-a
// udrey-hepburn "img/.pagination a/href" | humphrey -tmpl "{{.img|println}}" "img/.post-single-content img/src"

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
	toks := strings.SplitN(s, "/", 3)
	switch len(toks) {
	case 2:
		return &rule{toks[0], toks[1], ""}, nil
	case 3:
		return &rule{toks[0], toks[1], toks[2]}, nil
	}
	return nil, fmt.Errorf("can't parse rule: %s", s)
}

// apply the rule to the document and write the resulting array to map.
func (r *rule) apply(doc *goquery.Document, m map[string]interface{}) {
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

// download uses the http to download the page of url u
// and returns the results as an io.Reader
// It returns a non-nil error if downloading fails
// or the http response code is not 200
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

// downloadAndApplyRules tries to download the url u and apply the rules
// it return error if download or parsing fails
func downloadAndApplyRules(u string, rules []*rule) (map[string]interface{}, error) {
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
		rr.apply(doc, m)
	}

	return m, nil
}

var key = flag.String("key", "key", "the name for the url in output map")
var tmpl = flag.String("tmpl", "", "a text/template for output instead of json")
var page = flag.String("page", "", "the url to scrap")

func usage() {
	fmt.Fprintf(os.Stderr, "usage: humphrey [options] [rules]\n")
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

	if flag.NArg() == 0 {
		usage()
	}

	if *page == "" {
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
		enc.SetIndent("", "  ")
	}

	m, err := downloadAndApplyRules(*page, rules)
	if err != nil {
		log.Fatal(err)
	}
	m[*key] = *page


	if t != nil {
		if err := t.Execute(os.Stdout, m); err != nil {
			log.Fatal(err)
		}
	} else if enc != nil {
		if err := enc.Encode(m); err != nil {
			log.Fatal(err)
		}
	}
}
