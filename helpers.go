package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/html"
)

func getDBPool() (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DB_PATH")
	if dbURL == "" {
		return nil, fmt.Errorf("Couldn't get DB path from environment")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to database")

	return pool, nil
}

// Function to recursively traverse the HTML node tree
func findFeedLinks(n *html.Node, urls *[]SubscriptionTag) {
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
				*urls = append(*urls, SubscriptionTag{
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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":       user.Id,
			"username": user.Username,
			"email":    user.Email,
			"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
		})

	secret := []byte(os.Getenv("JWT_SECRET"))
	if secret == nil {
		return "", fmt.Errorf("Couldn't get JWT secret path from environment")
	}

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateJWT(tokenString string) (*Token, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Token{}, func(token *jwt.Token) (interface{}, error) {
		secret := []byte(os.Getenv("JWT_SECRET"))
		if secret == nil {
			return nil, fmt.Errorf("Couldn't get JWT secret path from environment")
		}
		// Ensure the signing method is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	// Get claims from token
	claims, ok := token.Claims.(*Token)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Get and check expiration
	if time.Now().Unix() > int64(claims.Exp) {
		return nil, fmt.Errorf("Token has expired")
	}

	return claims, nil
}
