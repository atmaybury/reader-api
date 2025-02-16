package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgx/v5"
	"golang.org/x/net/html"
)

func getDBConnection() (*pgx.Conn, error) {
	dbURL := "postgresql://andrew:WMI8fsHvYL0sR4hCOTGQ06zSxmoupIW9@dpg-cuo4sqrqf0us738rr4hg-a.singapore-postgres.render.com:5432/reader_db_z0oe"

	// conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to database")

	return conn, nil
}

// Function to recursively traverse the HTML node tree
func findFeedLinks(n *html.Node, urls *[]Subscription) {
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

func generateJWT(user User) (string, error) {
	secret := "temporarySecret"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":       user.Id,
			"username": user.Username,
			"email":    user.Email,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
