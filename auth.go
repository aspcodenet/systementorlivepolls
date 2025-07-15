package main

import (
	"net/url"

	"github.com/aspcodenet/systementorlivepolls/pages"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func WebPageAuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(pages.Userkey)
	var redirectUrl = url.QueryEscape("http://" + c.Request.Host + c.Request.RequestURI)
	if user == nil {
		c.Redirect(302, "/loginv1?redirect_uri="+redirectUrl)
		// Abort the request with the appropriate error code
		return
	}
	// Continue down the chain to handler etc
	c.Next()
}
