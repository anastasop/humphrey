
Humphrey is a simple scraper for html pages, intended to be used in shell scripts pipelines.

# Usage

The usage is very simple. It takes a list of css selectors, applies them to a url and outputs the results as json.

```
usage: humphrey [options] rules... url
rules:
  key/selector[/attribute]
options:
  -key string
       the name for the url in output map (default "key")
```

Each rule consists of 3 parts: key, css selector and optional attribute. Humphrey download the html of the url, parses it, applies the css selector and extracts the text of the elements matched or the text of the optional attribute. The output json object is of the form `{"key": values}` where `key` is the key of the rule and `values` is an array of strings for the matches. For example to get the list of repos (1st page) of the golang organization in github:

```
% humphrey repos/a.f4 http://github.com/golang
{
  "key": "https://github.com/golang",
  "repos": [
    "go",
    "tools",
    "build",
    "website",
    "geo",
    "vulndb",
    "telemetry",
    "perf",
    "crypto",
    "sync"
  ]
}
```

# Installation

`go install github.com/anastasop/humphrey@latest`

# TODO
1. Groups results of rules to a single key, for example it would be useful to select links and get in the same object like `{href: "", text, ""}`
