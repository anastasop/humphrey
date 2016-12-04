
Humphrey is a simple scraper for html pages, intended to be used in shell scripts pipelines.

# Usage

The usage is very simple. It reads a list of urls from stdin, extracts the required text as defined by the rules and outputs one json object for each urls scraped.

```
usage: humphrey [options] [rules]
rules:
  key:selector[:attribute]
options:
  -arrays
	Always store the result as array. Mostly useful with templates
  -key string
       the name for the url in output map (default "key")
  -page string
    	the url to scrap. If not set it reads all lines from stdin
  -pretty
	pretty print json
  -strict
	If a urls fails then stop the program (default true)
  -tmpl string
    	a text/template for output instead of json
```

Each rule consists of 3 parts: key, css selector and optional attribute. Humphrey download the html of a url, parses it, applies the css selector and extracts the text of the elements matched or the text of the optional attribute if specified. It then outputs the result as json. For example to get the names of all go packages:

```
humphrey -page http://golang.org/pkg "name:td.pkg-name>a"

{"key":"http://golang.org/pkg","name":["archive","tar","zip","bufio","builtin","bytes","compress","bzip2","flate","gzip","lzw","zlib","container","heap","list","ring","context","crypto","aes","cipher","des","dsa","ecdsa","elliptic","hmac","md5","rand","rc4","rsa","sha1","sha256","sha512","subtle","tls","x509","pkix","database","sql","driver","debug","dwarf","elf","gosym","macho","pe","plan9obj","encoding","ascii85","asn1","base32","base64","binary","csv","gob","hex","json","pem","xml","errors","expvar","flag","fmt","go","ast","build","constant","doc","format","importer","parser","printer","scanner","token","types","hash","adler32","crc32","crc64","fnv","html","template","image","color","palette","draw","gif","jpeg","png","index","suffixarray","io","ioutil","log","syslog","math","big","cmplx","rand","mime","multipart","quotedprintable","net","http","cgi","cookiejar","fcgi","httptest","httptrace","httputil","pprof","mail","rpc","jsonrpc","smtp","textproto","url","os","exec","signal","user","path","filepath","reflect","regexp","syntax","runtime","cgo","debug","msan","pprof","race","trace","sort","strconv","strings","sync","atomic","syscall","testing","iotest","quick","text","scanner","tabwriter","template","parse","time","unicode","utf16","utf8","unsafe"]}
```

The json object is of the form `{"key": values}` where `key` is the key of the rule and `values` the text of the elements matched. It can be `null`, a single string or an array of strings depending on how many elements matched. The option `arrays` enforces always an array with zero, one or many elements respectively.

For output a text/template can also be used. For example print in console all the titles for the index page o ycombinator
```
humphrey -tmpl "{{range .title}}{{.|println}}{{end}}" -page http://news.ycombinator.com "title:a.storylink"

The Mars Pathfinder website from 1997 is still online
Four Million Commutes Reveal New U.S. 'Megaregions'
Disassembling Sublime Text
TRust-DNS: implementing futures-rs and tokio-rs support
How I Wrote the Screenplay for “Arrival” and What I Learned Doing It
Magicians fought over an ultra-secret tracker dedicated to stealing magic tricks
```

Of course calls can be chained. Here is how to get all the image urls from a gallery of http://www.oldpicsarchive.com

```
humphrey -tmpl "{{.key|println}}{{range .img}}{{.|println}}{{end}}" -page http://www.oldpicsarchive.com/rare-photographs-vol-20-24-photos/ "img:.pagination a:href" | humphrey -tmpl "{{.img|println}}" "img:.post-single-content img:src"

http://www.oldpicsarchive.com/wp-content/uploads/2016/10/A-laboratory-technician-lifts-two-plastic-rods-from-a-boiling-bath-of-hot-sulfuric-acid-to-demonstrate-the-newly-invented-Teflon.-1940s.jpg
http://www.oldpicsarchive.com/wp-content/uploads/2016/10/50-years-ago-the-Welsh-mining-village-of-Aberfan-was-engulfed-by-a-coal-tip-landslide.-The-local-primary-school-was-directly-in-its-path.jpg
http://www.oldpicsarchive.com/wp-content/uploads/2016/10/Air-Rhodesia-Stewardesses-with-Friends-circa-1973.jpg
http://www.oldpicsarchive.com/wp-content/uploads/2016/10/1926-evening-gown-from-“Vogue”..jpg
http://www.oldpicsarchive.com/wp-content/uploads/2016/10/1960s.jpg
...
```

Prefer json when storing the results to an indexing service and use templates for shell scripts.

# Installation

`go get -u github.com/anastasop/humphrey`

# TODO
1. Download urls concurrently
2. Throttling downloader
3. Groups results of rules to a single key, for example it would be useful to select links and get in the same object `{href: "", text, ""}`

