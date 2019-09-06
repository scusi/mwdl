// mwdl.go - MalWareDownLoad is a go prog to fetch a (malware) file from a given URL via tor
//
// Examples
//
// Show available flags
//  mwdl -h
//
// Fetch a single URL (-u) via _tor_ (default), write output to local directory (default)
//  MalwareDownload -u=http://edenika.net/wp-content/plugins/cached_data/pdf_fax_message238413995.zip
//
// Fetch a singel URL useing a remote tor node
//  MalwareDownlaod -u=http://edenika.net/wp-content/plugins/cached_data/pdf_fax_message238413995.zip -t=tor.ccc.de:9050
//
// Note
//
// Files will be saved by the pattern `md5_of_url`_`filename_from_url`.
// Wheras 'md5_of_url' is the md5 sum of the url downloaded from and 'filename_from_url' is the filename extracted from the url.
// If the url does not contain a filename the generig string 'output' is beeing used instead.
//
package main

// import needed modules
import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"github.com/h12.me/socks"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var useragent string
var proxy string
var toraddr string
var urlstr string
var file string
var outDir string

func init() {
	flag.StringVar(&useragent, "ua", "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.1; WOW64; Trident/4.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; Media Center PC 6.0; .NET4.0C; .NET4.0E)", "user-agent to send with request")
	flag.StringVar(&proxy, "p", "", "addr of proxy to use")
	flag.StringVar(&toraddr, "t", "127.0.0.1:9050", "addr of tor node to use")
	flag.StringVar(&urlstr, "u", "", "url to fetch, if set 'f' is ignored")
	flag.StringVar(&file, "f", "", "file with urls to fetch, one per line")
	flag.StringVar(&outDir, "o", "./", "directory to write files to")
}

// getProxy - checks if a proxy is set via ENV HTTP_PROXY, http_proxy and returns that.
func getProxy(req *http.Request) (url *url.URL, err error) {
	if proxy != "" {
		u, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}
		return u, err
	} else {
		return nil, nil
	}
}

// checkErr - a function to check the error return value of another function.
func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
		os.Exit(2)
	}
}

// getContent - takes an url as argument and returns the body of the document
// from that url
func getContent(url string) (b []byte) {
	tr := &http.Transport{}
	if toraddr != "" {
		dialSocksProxy := socks.DialSocksProxy(socks.SOCKS5, toraddr)
		tr = &http.Transport{Dial: dialSocksProxy, Proxy: getProxy}
	} else {
		tr = &http.Transport{Proxy: getProxy}
	}
	client := &http.Client{Transport: tr}
	// generate request
	req, err := http.NewRequest("GET", url, nil)
	checkErr(err)

	if useragent != "none" {
		req.Header.Set("User-Agent", useragent)
	}

	resp, err := client.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)
	return body
}

// writeFile - takes an array of bytes, and a filename as argument, writes
// those bytes into a file with the supplied name (filename), returns the number of bytes written
func writeFile(b []byte, filename string) (i int) {
	f, err := os.Create(filename)
	checkErr(err)
	defer f.Close()
	i, err = f.Write(b)
	checkErr(err)
	return i
}

// filenameFromPath - takes a path (as a string) as argument,
// and returns the filename from that path
func filenameFromPath(path string) (filename string) {
	pathElements := strings.Split(path, "/")
	numOfElements := len(pathElements)
	filenameElement := pathElements[numOfElements-1 : numOfElements]
	filename = filenameElement[0]
	return filename
}

// fetchFromUrl - takes an url as argument (as string),
// retrieves the body (by calling getContent) from that url,
// parses the URL, extracts the filename from the parsed URL
// by calling filenameFromPath,
// writes the body to a file with the extracted filename and the md5 sum of the uri as prefix.
// logs howmany bytes have been written to the file (with the name
// extracted previous) to standard error.
func fetchFromUrl(uri string) string {
	l, err := url.Parse(uri)
	checkErr(err)
	// gen md5 sum of uri
	uriMd5 := md5.New()
	w := io.MultiWriter(uriMd5)
	w.Write([]byte(uri))
	// gen filename, if possible from uri
	var filename string
	if filenameFromPath(l.Path) == "" {
		filename = hex.EncodeToString(uriMd5.Sum(nil)) + "_" + "outfile"
	} else {
		filename = hex.EncodeToString(uriMd5.Sum(nil)) + "_" + filenameFromPath(l.Path)
	}
	// apply timestamp to filename
	ts := time.Now()
	tshort := ts.Format("2006-01-02-15-04-05-MST")
	filename = tshort + "_" + filename
	body := getContent(uri)
	// make sure outDir ends in a '/'
	if !strings.HasSuffix(outDir, "/") {
		outDir += "/"
	}
	i := writeFile(body, outDir+filename)
	log.Printf("%d bytes written to '%s'\n", i, filename)
	return filename
}

// gets the arguments 'fetch' was called with,
// iterates over arguments (urls) and calls fetchFromUrl for each of it.
func main() {
	flag.Parse()
	if urlstr != "" {
		fetchFromUrl(urlstr)
	}
	if file != "" {
		// handle file with urls
		fh, err := os.Open(file)
		checkErr(err)
		defer fh.Close()
		scanner := bufio.NewScanner(fh)
		for scanner.Scan() {
			// handle uri from scanner.Text()
			uri := scanner.Text()
			target := fetchFromUrl(uri)
			log.Printf("saved '%s' to '%s'\n", uri, target)
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}
