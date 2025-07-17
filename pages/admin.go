package pages

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aspcodenet/systementorlivepolls/data"
	"github.com/aspcodenet/systementorlivepolls/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RequestOption struct {
	DatabaseId uint   `json:"databaseId"`
	Text       string `json:"text"`
}

func AdminPollsSavePOST(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}

	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}

	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	var req struct {
		DatabaseId uint   `json:"databaseId"`
		Title      string `json:"title"`
		Questions  []struct {
			DatabaseId uint            `json:"databaseId"`
			Text       string          `json:"text"`
			Type       string          `json:"type"`
			Options    []RequestOption `json:"options"`
		} `json:"questions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var poll *data.Poll
	if req.DatabaseId == 0 {
		randString, _ := utils.RandString(16)
		poll = &data.Poll{
			Title:                req.Title,
			CurrentQuestionIndex: -1, // No question active yet
			Status:               "setup",
			AdminUserID:          int(adminUser.ID),
			InviteID:             randString,
		}
	} else {
		poll, _ = data.GetPollAndDetailsForAdmin(req.DatabaseId)
		poll.Title = req.Title

	}

	if poll.AdminUserID != int(adminUser.ID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to modify this poll."})
		return
	}

	for _, formQuestion := range req.Questions {
		// New or existing question?
		var updated = false
		if req.DatabaseId > 0 {
			for i, existingQuestion := range poll.Questions {
				if existingQuestion.ID == formQuestion.DatabaseId {
					poll.Questions[i].Text = formQuestion.Text
					poll.Questions[i].Type = formQuestion.Type
					poll.Questions[i].Options = syncOptions(poll.Questions[i].Options, formQuestion.Options)
					updated = true
					continue
				}
			}
		}
		if updated == false {
			newQuestion := data.Question{
				Text:  formQuestion.Text,
				Type:  formQuestion.Type,
				Votes: make(map[string]int), // Initialize empty map (will be populated from Vote table)
			}
			newQuestion.Options = syncOptions(newQuestion.Options, formQuestion.Options)
			poll.Questions = append(poll.Questions, newQuestion)
		}
	}

	// Save the poll and its associations to the database
	if result := data.DB.Save(&poll); result.Error != nil {
		log.Printf("Error creating poll in DB: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create poll in database."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pollId": fmt.Sprintf("%d", poll.ID), "message": "Poll saved successfully!"})

}

func syncOptions(fromDatabase []data.Option, fromForm []RequestOption) []data.Option {
	result := make([]data.Option, 0)

	// Map to track existing options
	existingOptions := make(map[uint]bool)

	// Add or update options from the form
	for _, formOption := range fromForm {
		found := false
		for i, dbOption := range fromDatabase {
			if formOption.DatabaseId == dbOption.ID {
				// Update existing option
				fromDatabase[i].Text = formOption.Text
				found = true
				break
			}
		}
		if !found {
			// Add new option
			result = append(result, data.Option{
				Text: formOption.Text,
			})
		}
		existingOptions[formOption.DatabaseId] = true
	}

	// Add options from database that weren't in the form
	// for _, dbOption := range fromDatabase {
	// 	if !existingOptions[dbOption.ID] {
	// 		result = append(result, dbOption)
	// 	}
	// }

	return result
}

func AdminPollsControlPanel(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}

	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}
	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	pollID := c.Param("inviteID")
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	//Here read the poll inclusive questions and inclusive options
	poll, err := data.GetPollWithDetails(pollID)
	if poll.AdminUserID != int(adminUser.ID) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	jsonData, _ := json.MarshalIndent(poll.Questions, "", "  ")

	c.HTML(http.StatusOK, "adminpollscontrolpanel.html", gin.H{
		"AdminUser": adminUser,
		"Poll":      poll,
		"AsJson":    string(jsonData),
	})

}

func AdminPollsCopyPOST(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}
	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
	}

	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	pollID, err := strconv.Atoi(c.Param("pollID"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	//Here read the poll inclusive questions and inclusive options
	poll, _ := data.GetPollAndDetailsForAdmin(uint(pollID))
	if poll.AdminUserID != int(adminUser.ID) {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}

	var count int64
	data.DB.Model(&data.Poll{}).Where("admin_user_id = ?", adminUser.ID).Count(&count)
	if count >= 5 {
		c.HTML(http.StatusOK, "maxpolls.html", gin.H{
			"AdminUser": adminUser,
		})
		return
	}

	//Here read the poll inclusive questions and inclusive options
	poll, _ = data.GetPollAndDetailsForAdmin(uint(pollID))
	if poll.AdminUserID != int(adminUser.ID) {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}
	//Copy poll and all details to new poll
	pollCopy := poll.DeepCopyWithoutID()
	pollCopy.Title = "Copy of " + poll.Title

	data.DB.Save(&pollCopy)
	c.Redirect(302, "/admin/polls/edit/"+strconv.Itoa(int(pollCopy.ID)))

}

func AdminPollsCopy(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}
	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
	}

	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	pollID, err := strconv.Atoi(c.Param("pollID"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	//Here read the poll inclusive questions and inclusive options
	poll, _ := data.GetPollAndDetailsForAdmin(uint(pollID))
	if poll.AdminUserID != int(adminUser.ID) {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}

	var count int64
	data.DB.Model(&data.Poll{}).Where("admin_user_id = ?", adminUser.ID).Count(&count)
	if count >= 5 {
		c.HTML(http.StatusOK, "maxpolls.html", gin.H{
			"AdminUser": adminUser,
		})
		return
	}

	c.HTML(http.StatusOK, "adminpollscopy.html", gin.H{
		"AdminUser": adminUser,
		"Poll":      poll,
	})

}

func AdminPollsDelete(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}
	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
	}

	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	pollID, err := strconv.Atoi(c.Param("pollID"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	//Here read the poll inclusive questions and inclusive options
	poll, _ := data.GetPollAndDetailsForAdmin(uint(pollID))
	if poll.AdminUserID != int(adminUser.ID) {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}

	c.HTML(http.StatusOK, "adminpollsdelete.html", gin.H{
		"AdminUser": adminUser,
		"Poll":      poll,
	})

}

func AdminPollsDeletePOST(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}
	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
	}

	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	pollID, err := strconv.Atoi(c.Param("pollID"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	//Here read the poll inclusive questions and inclusive options
	poll, _ := data.GetPollAndDetailsForAdmin(uint(pollID))
	if poll.AdminUserID != int(adminUser.ID) {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}

	if poll.Title != c.PostForm("name") {
		errors := make(map[string][]string)
		errors["Name"] = append(errors["Name"], "Poll name does not match")
		c.HTML(200, "adminpollsdelete.html", gin.H{
			"title":       "Delete poll",
			"CurrentUser": currentUser,
			"Poll":        poll,
			"errors":      errors,
		})
		return
	}

	data.DB.Delete(&poll)
	c.Redirect(302, "/admin/polls")

}

func AdminPollsEdit(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}

	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}
	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	pollID, err := strconv.Atoi(c.Param("pollID"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	//Here read the poll inclusive questions and inclusive options
	poll, err := data.GetPollAndDetailsForAdmin(uint(pollID))
	if poll.AdminUserID != int(adminUser.ID) {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}

	jsonData, _ := json.MarshalIndent(poll.Questions, "", "  ")

	c.HTML(http.StatusOK, "adminpollsedit.html", gin.H{
		"AdminUser": adminUser,
		"Poll":      poll,
		"AsJson":    string(jsonData),
	})

}

func AdminPollsNew(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}

	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}

	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	c.HTML(http.StatusOK, "adminpollsnew.html", gin.H{
		"AdminUser": adminUser,
	})
}

func AdminPolls(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	var currentUser = ""
	if user != nil {
		currentUser = user.(string)
	}

	if checkAdmin(currentUser) == false {
		c.HTML(http.StatusOK, "noadmin.html", gin.H{})
		return
	}

	var adminUser data.AdminUser
	err := data.DB.First(&adminUser, "email=?", currentUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(302, "/")
		return
	}

	dataObjects := []data.Poll{}
	if err := data.DB.Where("admin_user_id = ?", adminUser.ID).Find(&dataObjects).Error; err != nil && err != gorm.ErrRecordNotFound {
		c.AbortWithError(500, errors.New("Failed to retrieve polls"))
		return
	}

	q := c.Query("q")
	// Filter the dataObjects based on the query and type
	result := []data.Poll{}
	for _, db := range dataObjects {
		if q != "" {
			// Check if the database name contains the query string (case-insensitive)
			if !utils.ContainsIgnoreCase(db.Title, q) {
				continue
			}
		}
		result = append(result, db)
	}

	c.HTML(200, "adminpolls.html", gin.H{
		"title":       "Admin polls",
		"CurrentUser": currentUser,
		"q":           q,
		"Polls":       result,
	})

}

func checkAdmin(currentUser string) bool {
	adminsCSV := os.Getenv("ADMINS")
	if adminsCSV == "" {
		return true
	}
	adminEmails := strings.Split(adminsCSV, ",")
	// Check if the current user's email is in the list of admins
	for _, admin := range adminEmails {
		if strings.EqualFold(admin, currentUser) {
			return true
		}
	}
	return false
}
