package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Struct for getting slice of Twitch public RSA keys
type jwks_body struct {
	keys []map[string]string
}

// Struct for the Twitch "Get Users Follows" API response
/*
I need to find out what struct tags do(if anything) and why everything needs to be exported
*/
type Following_Body struct {
	Data       []Follow_Relationship `json:"data"`
	Count      int                   `json:"total"`
	Pagination map[string]string     `json:"pagination"`
}

type Follow_Relationship struct {
	From_Id     string `json:"from_id"`
	From_Login  string `json:"from_login"`
	From_Name   string `json:"from_name"`
	To_Id       string `json:"to_id"`
	To_Login    string `json:"to_login"`
	To_Name     string `json:"to_name"`
	Followed_At string `json:"followed_at"`
}

// Key lookup function
/*
This grabs the public RSA key from Twitch and returns it
*/
func lookupkey(kid interface{}) (interface{}, error) {
	jwks, err := http.Get("https://id.twitch.tv/oauth2/keys")

	if err != nil {
		panic(err)
	}

	var key map[string]string

	body := new(jwks_body)

	json.NewDecoder(jwks.Body).Decode(body)

	// If Twitch ever decides to have multiple public RSA keys this will get the right one
	for _, s := range body.keys {
		if s["kid"] == kid {
			key = s
			break
		}
	}

	return key, nil
}

// Get followers of user and return slice of maps
func get_following(user interface{}, auth string) []map[string]string {
	query := fmt.Sprintf("https://api.twitch.tv/helix/users/follows?from_id=%v", user)

	req, err := http.NewRequest("GET", query, nil)

	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", auth))
	req.Header.Add("Client-Id", "71emetuk00e580eacjw4o34dv5frsd")

	following_resp, err := http.DefaultClient.Do(req)

	if err != nil {
		panic(err)
	}

	defer following_resp.Body.Close()

	followings := new(Following_Body)

	json.NewDecoder(following_resp.Body).Decode(followings)

	var table []map[string]string

	for _, follow := range followings.Data {
		table = append(table, map[string]string{"channel": follow.To_Name, "followed_at": follow.Followed_At})
	}

	return table
}

func main() {

	// Load env file with client secret
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error Loading .env file")
	}

	router := gin.Default()

	router.Use(static.Serve("/assets", static.LocalFile("./assets", true)))
	router.LoadHTMLGlob("templates/*")

	var auth string
	// "Home" page (simple header and login button)
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "splash.tmpl", gin.H{
			"title": "Who do I follow on Twitch?",
			"link":  "https://id.twitch.tv/oauth2/authorize?client_id=71emetuk00e580eacjw4o34dv5frsd&redirect_uri=http://localhost:3000/logged_in&response_type=code&scope=openid",
		})
	})

	// Logged in page (gets ID token and then requests the user's follows from Twitch)
	router.GET("/logged_in", func(c *gin.Context) {

		token_request := url.Values{}
		token_request.Add("client_id", "71emetuk00e580eacjw4o34dv5frsd")
		token_request.Add("client_secret", os.Getenv("CLIENT_SECRET"))
		token_request.Add("code", c.Query("code"))
		token_request.Add("grant_type", "authorization_code")
		token_request.Add("redirect_uri", "http://localhost:3000/logged_in")

		resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", token_request)

		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body := new(map[string]string)

		json.NewDecoder(resp.Body).Decode(body)

		if err != nil {
			panic(err)
		}

		claims := jwt.MapClaims{}
		auth = (*body)["access_token"]

		_, err = jwt.ParseWithClaims((*body)["id_token"], claims, func(t *jwt.Token) (interface{}, error) {
			return lookupkey(t.Header["kid"])
		})

		user_id := claims["sub"]
		follow_table := get_following(user_id, auth)

		c.HTML(http.StatusOK, "loggedin.tmpl", gin.H{
			"title": fmt.Sprintf("Hello %v", claims["preferred_username"]),
			"table": follow_table,
		})
	})

	// Logout redirection is not working and upon navigating back to "/" and attempting to sign in there is no prompt to sign in (automatically signed in as user who just revoked access)
	router.GET("/logout", func(c *gin.Context) {
		logout_query := fmt.Sprintf("https://id.twitch.tv/oauth2/revoke?client_id=%v&token=%v", "71emetuk00e580eacjw4o34dv5frsd", auth)
		fmt.Println(logout_query)
		resp, logout_err := http.Post(logout_query, "", nil)

		fmt.Println(resp.Status)

		if logout_err != nil {
			panic(logout_err)
		}

		if resp.StatusCode != 200 {
			panic(resp.StatusCode)
		}

		auth = ""

		c.Request.URL.Path = "/"
		router.HandleContext(c)
	})

	// Currently set for localhost:3000
	router.Run(":3000")
}
