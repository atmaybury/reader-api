package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Subscription struct {
	title string
	href  string
}

// Sends 200
func handleRoot(w http.ResponseWriter, r *http.Request) {}

// Given a URL, find any rss links and save them
func handleAddSubscription(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf("Method not allowed: %v", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Get URL for querying
	inputURL := r.URL.Query().Get("url")
	if inputURL == "" {
		http.Error(w, fmt.Sprintf("Missing url parameter: %v", r.Method), http.StatusBadRequest)
		return
	}

	// Validate url param
	parsedURL, err := url.ParseRequestURI(inputURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// Make GET request to the URL
	resp, err := http.Get(parsedURL.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error making GET request to %s: %v", parsedURL, err), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Received %d response", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	// Parse html
	doc, err := html.Parse(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing html response: %v", err), http.StatusInternalServerError)
		return
	}

	// slice of rss urls
	var urls []Subscription

	// Traverse the HTML document
	findFeedLinks(doc, &urls)

	if len(urls) == 0 {
		http.Error(w, fmt.Sprint("No feed URLs found"), http.StatusInternalServerError)
		return
	}

	// TODO save links
	for _, url := range urls {
		fmt.Println("----------")
		fmt.Println(url.title)
		fmt.Println(url.href)
		fmt.Println("----------")
	}
}

// Function to recursively traverse the HTML node tree
func findFeedLinks(n *html.Node, urls *[]Subscription) {
	fmt.Printf("%v %v\n", n.Type, n.Data)
	if n.Type == html.ElementNode && n.Data == "link" {
		var rel, attrtype, title, href string
		for _, attr := range n.Attr {
			switch attr.Key {
			case "rel":
				rel = attr.Val
			case "type":
				attrtype = attr.Val
			case "title":
				title = attr.Val
			case "href":
				href = attr.Val
			}
		}

		// Check if the link is an RSS or Atom feed
		if strings.ToLower(rel) == "alternate" {
			if strings.Contains(
				strings.ToLower(attrtype), "rss") || strings.Contains(strings.ToLower(attrtype), "atom") {
				*urls = append(*urls, Subscription{
					title: title,
					href:  href,
				})
			}
		}
	}

	// Recursively traverse child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		findFeedLinks(c, urls)
	}
}
