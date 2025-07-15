package pages

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/aspcodenet/systementorlivepolls/data"
	"github.com/aspcodenet/systementorlivepolls/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var GithubClientID string
var GithubClientSecret string
var Userkey string = "theuserKey"

func Init(githubClientID, githubClientSecret string) {
	GithubClientID = githubClientID
	GithubClientSecret = githubClientSecret
}

func Start(c *gin.Context) {

	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}
	rootHTML := `
	<h1>My web app</h1>
	<p>Using raw HTTP OAuth 2.0</p>
	<p>You can log into this app with your GitHub credentials:</p>
	<p><a href="/login">Log in with GitHub</a></p>`
	rootHTML += "<p>Logged in as: " + currentUser + "</p>"

	c.HTML(http.StatusOK, "home.html", gin.H{
		"title":       "Systementor Live pollsI",
		"text":        rootHTML,
		"CurrentUser": currentUser,
	})
	//c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(rootHTML))
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(302, "/")
}

func GithubLoginHandler(c *gin.Context) {
	// Step 1: Request a user's GitHub identity
	//
	// ... by redirecting the user's browser to a GitHub login endpoint. We're not
	// setting redirect_uri, leaving it to GitHub to use the default we set for
	// this application: /github/callback
	// We're also not asking for any specific scope, because we only need access
	// to the user's public information to know that the user is really logged in.
	//
	// We're setting a random state cookie for the client to return
	// to us when the call comes back, to prevent CSRF per
	// section 10.12 of https://www.rfc-editor.org/rfc/rfc6749.html
	state, err := utils.RandString(16)
	if err != nil {
		panic(err)
	}

	//c.Request.AddCookie(cookie)
	session := sessions.Default(c)
	session.Set("STATE-LOGIN", state)
	session.Save()
	//http.SetCookie(c.Writer, cookie)

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?scope=user:email&client_id=%s&state=%s", GithubClientID, state)

	http.Redirect(c.Writer, c.Request, redirectURL, 302)
}

func GithubCallbackHandler(c *gin.Context) {
	// Step 2: Users are redirected back to your site by GitHub
	//
	// The user is authenticated w/ GitHub by this point, and GH provides us
	// a temporary code we can exchange for an access token using the app's
	// full credentials.
	//
	// Start by checking the state returned by GitHub matches what
	// we've stored in the cookie.
	session := sessions.Default(c)
	state := session.Get("STATE-LOGIN")
	if c.Request.URL.Query().Get("state") != state {
		http.Error(c.Writer, "state did not match", http.StatusBadRequest)
		return
	}

	// We use the code, alongside our client ID and secret to ask GH for an
	// access token to the API.
	code := c.Request.URL.Query().Get("code")
	requestBodyMap := map[string]string{
		"client_id":     GithubClientID,
		"client_secret": GithubClientSecret,
		"code":          code,
	}
	requestJSON, err := json.Marshal(requestBodyMap)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(requestJSON))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(c.Writer, "unable to connect to access_token endpoint"+err.Error(), http.StatusInternalServerError)
		return
	}
	respbody, _ := io.ReadAll(resp.Body)

	fmt.Println("START respnody")
	fmt.Println(string(respbody))
	fmt.Println("END")

	// Represents the response received from Github
	var ghresp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	json.Unmarshal(respbody, &ghresp)

	fmt.Println("START")
	fmt.Println(ghresp.AccessToken)
	fmt.Println("END")
	// Step 3: Use the access token to access the API
	//
	// With the access token in hand, we can access the GitHub API on behalf
	// of the user. Since we didn't provide a scope, we only get access to
	// the user's public information.
	email := getGitHubUserInfo(ghresp.AccessToken)

	//session = sessions.Default(c)
	session.Set(Userkey, email)
	session.Save()

	var adminUser data.AdminUser
	err = data.DB.First(&adminUser, "email=?", email).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		adminUser.Email = email
		data.DB.Create(&adminUser)
	}

	redirectUrl := c.DefaultQuery("redirect_uri", "/admin/polls")
	c.Redirect(302, redirectUrl)
}

func getGitHubUserInfo(accessToken string) string {
	// Query the GH API for user info
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	respbody, _ := io.ReadAll(resp.Body)

	fmt.Println("START")
	fmt.Println(respbody)
	fmt.Println("END")

	var result []map[string]interface{}
	json.Unmarshal([]byte(respbody), &result)

	//email := result[0]["email"]
	email := result[0]["email"]
	fmt.Println("Email: ", email)

	return email.(string)
}
