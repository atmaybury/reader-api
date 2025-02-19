package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgx/v5"
	"golang.org/x/net/html"
)

func getDBConnection() (*pgx.Conn, error) {
	dbURL, exists := os.LookupEnv("DB_PATH")
	if !exists {
		return nil, fmt.Errorf("Couldn't get DB path from environment")
	}

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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":       user.Id,
			"username": user.Username,
			"email":    user.Email,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	secret, exists := os.LookupEnv("JWT_SECRET")
	if !exists {
		return "", fmt.Errorf("Couldn't get JWT secret path from environment")
	}

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateJWT(tokenString string) (*jwt.MapClaims, error) {
	secret, exists := os.LookupEnv("JWT_SECRET")
	if !exists {
		return nil, fmt.Errorf("Couldn't get JWT secret path from environment")
	}

	// Parse the token with the secret key
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			// Ensure the signing method is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret, nil
		},
	)
	if err != nil {
		return nil, err
	}

	// Check if token is valid
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Check expiration
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return nil, fmt.Errorf("token has expired")
			}
		} else {
			return nil, fmt.Errorf("invalid expiration claim")
		}

		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
