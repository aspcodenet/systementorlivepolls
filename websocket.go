package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/aspcodenet/systementorlivepolls/data"
	"github.com/aspcodenet/systementorlivepolls/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// import (
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"strconv"
// 	"sync"

// 	"github.com/aspcodenet/systementorlivepolls/data"
// 	"github.com/gin-gonic/gin"
// 	"github.com/gorilla/websocket"
// )

var (
	// connections stores WebSocket connections, keyed by PollID, then by connection pointer.
	connections = make(map[string]map[*websocket.Conn]bool)
	// mutex for protecting access to connections map. Polls are now managed via DB and their internal mutex.
	globalMutex = &sync.Mutex{}

	// WebSocket upgrader.
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for simplicity in this example
		},
	}
)

type WebSocketMessage struct {
	Type   string `json:"type"` // e.g., "submit_vote", "admin_action", "poll_state_update", "admin_results_update"
	PollID string `json:"pollId"`
	Status string `json:"status,omitempty"`

	QuestionID      string                    `json:"questionId,omitempty"`
	SelectedOptions []string                  `json:"selectedOptions,omitempty"` // For user votes
	Action          string                    `json:"action,omitempty"`          // For admin actions: "next", "show_results", "done"
	CurrentQuestion *data.Question            `json:"currentQuestion,omitempty"` // For poll state updates
	Results         map[string]map[string]int `json:"results,omitempty"`         // For poll state updates (overall results)
	Votes           map[string]int            `json:"votes,omitempty"`           // For admin results update (current question votes)
	TotalVotes      int                       `json:"totalVotes,omitempty"`      // For admin results update
	Message         string                    `json:"message,omitempty"`
	AllQuestions    []data.Question           `json:"allQuestions,omitempty"` // For final results, includes all question details
	VoterID         string                    `json:"voterId,omitempty"`      // Identifier for the voter
}

func handleWebSocket(c *gin.Context) {
	pollIDStr := c.Param("inviteID")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade websocket for poll %s: %v", pollIDStr, err)
		return
	}
	defer conn.Close()

	// Retrieve poll from DB
	p, err := data.GetPollWithDetails(pollIDStr)
	if err != nil {
		log.Printf("WebSocket connection attempted for non-existent poll or DB error: %s, %v", pollIDStr, err)
		conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Poll does not exist or internal error."})
		return
	}

	// Add connection to global map
	globalMutex.Lock()
	if _, ok := connections[pollIDStr]; !ok {
		connections[pollIDStr] = make(map[*websocket.Conn]bool)
	}
	connections[pollIDStr][conn] = true
	globalMutex.Unlock()

	log.Printf("Client connected to poll %s via WebSocket.", pollIDStr)

	// Send initial poll state to the newly connected client
	p.Mu.Lock() // Lock the specific poll's mutex
	initialStateMsg := getPollStateMessage(p)
	p.Mu.Unlock()
	if err := conn.WriteJSON(initialStateMsg); err != nil {
		log.Printf("Error sending initial state to new client for poll %s: %v", pollIDStr, err)
	}

	// If admin, send initial real-time results
	if c.Request.URL.Query().Get("role") == "admin" {
		p.Mu.Lock() // Lock the specific poll's mutex
		adminResultsMsg := getAdminResultsUpdateMessage(p)
		p.Mu.Unlock()
		if err := conn.WriteJSON(adminResultsMsg); err != nil {
			log.Printf("Error sending initial admin results to new client for poll %s: %v", pollIDStr, err)
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Client disconnected from poll %s.", pollIDStr)
			} else {
				log.Printf("Error reading message from websocket for poll %s: %v", pollIDStr, err)
			}
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshalling message for poll %s: %v", pollIDStr, err)
			continue
		}

		// Re-fetch the poll from DB to ensure we have the latest state before modifying
		// This is important for concurrent access and data integrity.
		p, _ := data.GetPollWithDetails(pollIDStr)
		if err != nil {
			log.Printf("Error re-fetching poll %s from DB during WebSocket message processing: %v", pollIDStr, err)
			conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Internal server error: poll data unavailable."})
			continue
		}

		p.Mu.Lock() // Lock the specific poll for modifications
		switch msg.Type {
		case "submit_vote":
			if p.Status != "active" {
				log.Printf("Vote submitted for poll %s when not active. Status: %s", pollIDStr, p.Status)
				conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Voting is not currently active."})
				p.Mu.Unlock()
				continue
			}
			if p.CurrentQuestionIndex == -1 || p.CurrentQuestionIndex >= len(p.Questions) {
				log.Printf("Vote submitted for poll %s with no active question.", pollIDStr)
				conn.WriteJSON(WebSocketMessage{Type: "error", Message: "No active question to vote on."})
				p.Mu.Unlock()
				continue
			}

			currentQ := &p.Questions[p.CurrentQuestionIndex]
			// The client sends `questionId` as a string, convert it to uint for comparison.
			clientQID, parseErr := strconv.ParseUint(msg.QuestionID, 10, 32)
			if parseErr != nil || uint(clientQID) != currentQ.ID {
				log.Printf("Vote submitted for wrong question ID. Expected GORM ID %d, got %s", currentQ.ID, msg.QuestionID)
				conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Invalid question for voting."})
				p.Mu.Unlock()
				continue
			}

			// Get or generate VoterID
			voterID := msg.VoterID
			if voterID == "" {
				// If client doesn't provide a VoterID, generate a simple unique string for this vote.
				// In a real application, you would manage user sessions or persistent IDs (e.g., via cookies).
				generatedID, err := utils.RandString(16) // 16 bytes for a decent length
				if err != nil {
					log.Printf("Error generating voter ID: %v", err)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Failed to establish voter session."})
					p.Mu.Unlock()
					return // Terminate connection if ID cannot be generated
				}
				voterID = generatedID
				log.Printf("Generated temporary voter ID: %s", voterID)
			}

			// Validate selected options against the options stored in the current question
			validOptionsMap := make(map[uint]bool) // Use uint for option IDs
			for _, opt := range currentQ.Options {
				validOptionsMap[opt.ID] = true
			}

			var selectedOptionIDs []uint
			for _, selectedOptIDStr := range msg.SelectedOptions {
				optID, err := strconv.ParseUint(selectedOptIDStr, 10, 32) // Parse to uint
				if err != nil {
					log.Printf("Invalid option ID format received: %s, error: %v", selectedOptIDStr, err)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: fmt.Sprintf("Invalid option ID format: %s", selectedOptIDStr)})
					p.Mu.Unlock()
					continue
				}
				if !validOptionsMap[uint(optID)] {
					log.Printf("Invalid option ID submitted: %s", selectedOptIDStr)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: fmt.Sprintf("Invalid option ID: %s", selectedOptIDStr)})
					p.Mu.Unlock()
					continue
				}
				selectedOptionIDs = append(selectedOptionIDs, uint(optID))
			}

			if currentQ.Type == "single-select" {
				if len(selectedOptionIDs) > 1 {
					log.Printf("Multiple options selected for single-select question.")
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Please select only one option for this question."})
					p.Mu.Unlock()
					return
				}
				// For single-select, remove previous votes by this voter for this question
				if result := data.DB.Where("question_id = ? AND voter_id = ?", currentQ.ID, voterID).Delete(&data.Vote{}); result.Error != nil {
					log.Printf("Error deleting previous votes for single-select: %v", result.Error)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Failed to update vote."})
					p.Mu.Unlock()
					return
				}
			}

			// Create new vote records
			for _, selectedOptID := range selectedOptionIDs {
				newVote := data.Vote{
					QuestionID: currentQ.ID,
					OptionID:   selectedOptID,
					VoterID:    voterID,
				}
				if result := data.DB.Create(&newVote); result.Error != nil {
					log.Printf("Error saving vote to DB: %v", result.Error)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Failed to save vote."})
					p.Mu.Unlock()
					return
				}
			}
			log.Printf("Vote(s) received for poll %s, question %d by voter %s", pollIDStr, currentQ.ID, voterID)

			// Notify admin of real-time vote update
			broadcastMessage(pollIDStr, getAdminResultsUpdateMessage(p))

		case "admin_action":
			log.Printf("Admin action received for poll %s: %s", pollIDStr, msg.Action)
			switch msg.Action {
			case "start":
				if p.Status == "setup" && len(p.Questions) > 0 {
					p.CurrentQuestionIndex = 0
					p.Status = "active"
					log.Printf("Admin started poll %s. Moving to question %d.", pollIDStr, p.CurrentQuestionIndex+1)
					if result := data.DB.Save(p); result.Error != nil { // Save poll status and index
						log.Printf("Error saving poll status to DB: %v", result.Error)
						conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Failed to start poll."})
						p.Mu.Unlock()
						continue
					}
					broadcastMessage(pollIDStr, getPollStateMessage(p))
					broadcastMessage(pollIDStr, getAdminResultsUpdateMessage(p)) // Send initial results to admin
				} else {
					log.Printf("Admin tried to start poll %s, but status is %s or no questions.", pollIDStr, p.Status)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Cannot start poll. Ensure questions are added and poll is in 'setup' status."})
				}
			case "next":
				if p.Status == "active" || p.Status == "results" {
					p.CurrentQuestionIndex++
					if p.CurrentQuestionIndex < len(p.Questions) {
						p.Status = "active" // Move to next question, set status back to active
						log.Printf("Admin moved poll %s to next question %d.", pollIDStr, p.CurrentQuestionIndex+1)
						if result := data.DB.Save(p); result.Error != nil { // Save poll status and index
							log.Printf("Error saving poll status to DB: %v", result.Error)
							conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Failed to move to next question."})
							p.Mu.Unlock()
							continue
						}
						broadcastMessage(pollIDStr, getPollStateMessage(p))
						broadcastMessage(pollIDStr, getAdminResultsUpdateMessage(p)) // Reset admin results for new question
					} else {
						p.Status = "finished"
						log.Printf("Admin finished poll %s. All questions answered.", pollIDStr)
						if result := data.DB.Save(p); result.Error != nil { // Save poll status
							log.Printf("Error saving poll status to DB: %v", result.Error)
							conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Failed to finish poll."})
							p.Mu.Unlock()
							continue
						}
						broadcastMessage(pollIDStr, getPollStateMessage(p))
					}
				} else {
					log.Printf("Admin tried to move poll %s to next, but status is %s.", pollIDStr, p.Status)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Cannot move to next question. Poll is not active or in results mode."})
				}
			case "show_results":
				if p.Status == "active" {
					p.Status = "results"
					log.Printf("Admin showed results for poll %s, question %d.", pollIDStr, p.CurrentQuestionIndex+1)
					if result := data.DB.Save(p); result.Error != nil { // Save poll status
						log.Printf("Error saving poll status to DB: %v", result.Error)
						conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Failed to show results."})
						p.Mu.Unlock()
						continue
					}
					broadcastMessage(pollIDStr, getPollStateMessage(p))
				} else {
					log.Printf("Admin tried to show results for poll %s, but status is %s.", pollIDStr, p.Status)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Cannot show results. Poll is not active."})
				}
			case "done": // This action signifies the end of the entire poll
				p.Status = "finished"
				log.Printf("Admin marked poll %s as done. Final results displayed.", pollIDStr)
				if result := data.DB.Save(p); result.Error != nil { // Save poll status
					log.Printf("Error saving poll status to DB: %v", result.Error)
					conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Failed to mark poll as done."})
					p.Mu.Unlock()
					continue
				}
				broadcastMessage(pollIDStr, getPollStateMessage(p))
			default:
				log.Printf("Unknown admin action: %s", msg.Action)
				conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Unknown admin action."})
			}
		default:
			log.Printf("Unknown message type received for poll %s: %s", pollIDStr, msg.Type)
			conn.WriteJSON(WebSocketMessage{Type: "error", Message: "Unknown message type."})
		}
		p.Mu.Unlock() // Unlock the specific poll after modifications
	}

	// Clean up connection on disconnect
	globalMutex.Lock()
	delete(connections[pollIDStr], conn)
	if len(connections[pollIDStr]) == 0 {
		delete(connections, pollIDStr) // Clean up poll entry if no connections left
	}
	globalMutex.Unlock()
}

func getPollStateMessage(p *data.Poll) WebSocketMessage {
	msg := WebSocketMessage{
		Type:   "poll_state_update",
		PollID: fmt.Sprintf("%d", p.ID), // Convert uint ID to string for WebSocketMessage
		Status: p.Status,
	}

	// Load votes for all questions if status is results or finished, or for current question if active
	if p.Status == "active" || p.Status == "results" {
		if p.CurrentQuestionIndex >= 0 && p.CurrentQuestionIndex < len(p.Questions) {
			currentQ := &p.Questions[p.CurrentQuestionIndex] // Get a pointer to modify the struct in the slice
			// Populate Votes map for current question
			votes, err := getVotesForQuestion(currentQ.ID)
			if err != nil {
				log.Printf("Error getting votes for current question %d: %v", currentQ.ID, err)
				// In a real app, you might send an error message to the client or handle gracefully
			} else {
				currentQ.Votes = votes
			}
			msg.CurrentQuestion = currentQ
		} else {
			log.Printf("DEBUG Go: CurrentQuestionIndex out of bounds for poll %d. Index: %d, Questions count: %d",
				p.ID, p.CurrentQuestionIndex, len(p.Questions))
		}
	}

	if p.Status == "results" || p.Status == "finished" {
		allResults := make(map[string]map[string]int)
		var allQuestionsForMsg []data.Question
		for i := range p.Questions { // Iterate by index to get mutable question
			q := &p.Questions[i]
			votes, err := getVotesForQuestion(q.ID)
			if err != nil {
				log.Printf("Error getting votes for question %d: %v", q.ID, err)
				// In a real app, you might send an error message to the client or handle gracefully
			} else {
				q.Votes = votes // Populate the transient Votes map
			}

			allResults[fmt.Sprintf("%d", q.ID)] = make(map[string]int)
			for optionID, count := range q.Votes {
				allResults[fmt.Sprintf("%d", q.ID)][optionID] = count
			}
			allQuestionsForMsg = append(allQuestionsForMsg, *q) // Append a copy
		}
		msg.Results = allResults
		msg.AllQuestions = allQuestionsForMsg
	}
	return msg
}

func getVotesForQuestion(questionID uint) (map[string]int, error) {
	var votes []data.Vote
	// Fetch all votes for the given question
	result := data.DB.Where("question_id = ?", questionID).Find(&votes)
	if result.Error != nil {
		return nil, result.Error
	}

	voteCounts := make(map[string]int)
	for _, vote := range votes {
		voteCounts[fmt.Sprintf("%d", vote.OptionID)]++ // Use OptionID as string key
	}

	var options []data.Option
	result = data.DB.Where("question_id = ?", questionID).Find(&options)
	if result.Error != nil {
		return nil, result.Error
	}
	for _, option := range options {
		// Check if the option exists in the voteCounts map
		if _, exists := voteCounts[fmt.Sprintf("%d", option.ID)]; !exists {
			voteCounts[fmt.Sprintf("%d", option.ID)] = 0 // Initialize count to zero if not present
		}

	}

	return voteCounts, nil
}

func getAdminResultsUpdateMessage(p *data.Poll) WebSocketMessage {
	msg := WebSocketMessage{
		Type:   "admin_results_update",
		PollID: fmt.Sprintf("%d", p.ID),
	}

	if p.CurrentQuestionIndex >= 0 && p.CurrentQuestionIndex < len(p.Questions) {
		currentQ := &p.Questions[p.CurrentQuestionIndex]
		votes, err := getVotesForQuestion(currentQ.ID)
		if err != nil {
			log.Printf("Error getting votes for admin results update for question %d: %v", currentQ.ID, err)
		} else {
			currentQ.Votes = votes
		}

		msg.QuestionID = fmt.Sprintf("%d", currentQ.ID)
		msg.Votes = currentQ.Votes
		total := 0
		for _, count := range currentQ.Votes {
			total += count
		}
		msg.TotalVotes = total
	}
	return msg
}

func broadcastMessage(pollID string, msg WebSocketMessage) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	conns, ok := connections[pollID]
	if !ok {
		log.Printf("No connections found for poll ID: %s", pollID)
		return
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshalling message: %v", err)
		return
	}

	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			log.Printf("Error writing message to websocket for poll %s: %v", pollID, err)
			conn.Close()
			delete(conns, conn) // Remove broken connection
		}
	}
}
