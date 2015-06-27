package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

// This is the communication channel that
// each Crawl will use to determine whether or not
// the Url needs to be fetched again
type Examine struct {
	Goahead chan bool // This is a private channel between each go
	                  // routine of Crawl
	Url     string	// This is the Url that each go routine of Crawl
	                // must ask the examine channel
}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
// examine => this is the single global channel that will control whether
//            the said Url is to be crawled again
// ch -> this is the concurrent channel for the enclosing routine
func Crawl(url string, depth int, fetcher Fetcher, examine chan Examine, ch chan string) {
	// Use defer to ensure the channel for concurrent control is always talked to
	defer func () { ch <- url}()
	//fmt.Printf("    Running %v\n", url)
	
	// Talk to the examine channel to determine whether or not
	//   my Url should be processed again
	// Since we made the examine channel a global channel, this
	//   examination should be thread safe
	shoulddo := make(chan bool)
	examine <- Examine{shoulddo, url}
	b := <-shoulddo
	//fmt.Printf("    %v for %v\n", b, url)
	if !b {
		//fmt.Printf("    Would not fetch %v\n", url)
		return
	}
	if depth <= 0 {
		return
	}
	// The global controller has given the go ahead, let's
	//   fetch the url
	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("found: %s %q\n", url, body)
	
	// For each child, open a channel for concurrent control
	subch := make(chan string)
	for _, u := range urls {
		go Crawl(u, depth-1, fetcher, examine, subch)
	}
	// Wait for all the children to complete
	for range urls {
		<-subch
		//fmt.Printf("    Done %v\n", <-subch)
	}
	return
}

func main() {
	// Create a global examine channel that we could control
	//   the Url uniqueness (or any other examination that require
	//   a centralize/synchronized read/write)
	examine := make(chan Examine)
	
	// This is the concurrent channel, for this instance,
	//   this will only be waiting on one child, the initial link
	ch := make(chan string)
	
	// Here is the map that is needed for us to determine whether
	//   or not an URL should be traverse again
	var emap = make(map[string]bool)
	go Crawl("http://golang.org/", 4, fetcher, examine, ch)
	
	// Use a go function to examine the Url based on the emap that
	//   this controls
	// Simply note down whether or not a given Url is in the map, the
	//    communicate the go head signal to the examined instance
	go func() {
		for {
			v := <-examine
			goahead := emap[v.Url]
			emap[v.Url] = true
			v.Goahead <- !goahead
		}
	}()
	
	// Finally, wait for the lead crawl to complete
	<-ch
}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"http://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"http://golang.org/pkg/",
			"http://golang.org/cmd/",
		},
	},
	"http://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"http://golang.org/",
			"http://golang.org/cmd/",
			"http://golang.org/pkg/fmt/",
			"http://golang.org/pkg/os/",
		},
	},
	"http://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"http://golang.org/",
			"http://golang.org/pkg/",
		},
	},
	"http://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"http://golang.org/",
			"http://golang.org/pkg/",
		},
	},
}
