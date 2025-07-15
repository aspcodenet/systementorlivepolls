package data

import (
	"fmt"
	"log"
	"sync"

	"gorm.io/gorm"
)

type Option struct {
	gorm.Model        // Adds ID, CreatedAt, UpdatedAt, DeletedAt
	Text       string `json:"text"`
	QuestionID uint   `json:"-" gorm:"index"` // Foreign key to Question, '-' to ignore in JSON marshal
}

// Question represents a single question in a poll.
type Question struct {
	gorm.Model          // Adds ID, CreatedAt, UpdatedAt, DeletedAt
	Text       string   `json:"text"`
	Type       string   `json:"type"`                                 // "single-select" or "multi-select"
	Options    []Option `json:"options" gorm:"foreignKey:QuestionID"` // One-to-many relationship
	PollID     uint     `json:"-" gorm:"index"`                       // Foreign key to Poll

	// Transient field for votes, not stored directly by GORM but populated from Vote table
	Votes map[string]int `json:"votes" gorm:"-"` // '-' to ignore by GORM, handled manually
}

// Poll represents the entire poll.
type Poll struct {
	gorm.Model                      // Adds ID, CreatedAt, UpdatedAt, DeletedAt
	Title                string     `json:"title"`
	Questions            []Question `json:"questions" gorm:"foreignKey:PollID"` // One-to-many relationship
	CurrentQuestionIndex int        `json:"currentQuestionIndex"`
	Status               string     `json:"status"` // "setup", "active", "results", "finished"
	AdminUserID          int        `json:"-"`
	mu                   sync.Mutex `gorm:"-"` // Mutex for protecting poll data, ignore by GORM
}

// Vote represents a single vote by a user for an option.
type Vote struct {
	gorm.Model
	QuestionID uint   `gorm:"index"` // Foreign key to Question
	OptionID   uint   `gorm:"index"` // Foreign key to Option
	VoterID    string `gorm:"index"` // Identifier for the voter (e.g., session ID, user ID)
}

// GetPollWithDetails retrieves a poll, its questions, options, and aggregates vote counts.
func GetPollWithDetails(pollID uint) (*Poll, error) {
	poll := &Poll{}

	// 1. Eager load the Poll with its Questions and nested Options
	//    Preload("Questions.Options") tells GORM to load Poll -> Questions -> Options
	err := DB.Preload("Questions.Options").First(poll, pollID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("poll with ID %d not found", pollID)
		}
		return nil, fmt.Errorf("failed to retrieve poll and its questions/options: %w", err)
	}

	// 2. Aggregate Votes for all questions in the poll
	questionIDs := []uint{}
	for _, q := range poll.Questions {
		questionIDs = append(questionIDs, q.ID)
	}

	// Use a struct to hold the raw aggregated vote data
	// Note: gorm.Model is optional here, we just need the fields
	type VoteAggregationResult struct {
		QuestionID uint
		OptionID   uint
		Count      int `gorm:"column:count"` // Alias for the COUNT(*) result
	}

	var rawVoteCounts []VoteAggregationResult
	if len(questionIDs) > 0 { // Only query if there are questions
		err = DB.
			Table("votes"). // Specify the table name
			Select("question_id, option_id, COUNT(*) as count").
			Where("question_id IN (?)", questionIDs).
			Group("question_id, option_id").
			Find(&rawVoteCounts).Error
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate votes: %w", err)
		}
	}

	// 3. Populate the 'Votes' map in each Question struct
	//    First, create lookup maps for quick access
	questionMap := make(map[uint]*Question)
	for i := range poll.Questions {
		q := &poll.Questions[i]        // Get pointer to modify in slice
		q.Votes = make(map[string]int) // Initialize the map
		questionMap[q.ID] = q
	}

	optionMap := make(map[uint]string) // Map OptionID to Option.Text
	for _, q := range poll.Questions {
		for _, opt := range q.Options {
			optionMap[opt.ID] = opt.Text
		}
	}

	// Iterate through the aggregated vote counts and populate the Question.Votes map
	for _, va := range rawVoteCounts {
		if q, ok := questionMap[va.QuestionID]; ok {
			if optionText, ok := optionMap[va.OptionID]; ok {
				q.Votes[optionText] = va.Count
			} else {
				log.Printf("Warning: OptionID %d not found for QuestionID %d, skipping vote aggregation for this option.", va.OptionID, va.QuestionID)
			}
		}
	}

	return poll, nil
}
