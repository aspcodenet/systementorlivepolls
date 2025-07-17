package data

import (
	"github.com/aspcodenet/systementorlivepolls/utils"
	"gorm.io/gorm"
)

func (p *Poll) DeepCopyWithoutID() *Poll {
	if p == nil {
		return nil
	}
	inviteID, _ := utils.RandString(16)
	newPoll := &Poll{
		Title:                p.Title,
		CurrentQuestionIndex: p.CurrentQuestionIndex,
		Status:               p.Status,
		AdminUserID:          p.AdminUserID,
		// Explicitly set ID to 0. Copy other gorm.Model fields.
		Model: gorm.Model{
			ID:        0, // ID is reset to 0
			CreatedAt: p.Model.CreatedAt,
			UpdatedAt: p.Model.UpdatedAt,
			DeletedAt: p.Model.DeletedAt,
		},
		InviteID: inviteID,
		// Mutex 'mu' should not be copied. A new object should have its own mutex
		// initialized to its zero value, which is safe.
	}

	// Deep copy the Questions slice
	if p.Questions != nil {
		newPoll.Questions = make([]Question, len(p.Questions))
		for i, q := range p.Questions {
			// Recursively call DeepCopyWithoutID for each Question
			newPoll.Questions[i] = *q.DeepCopyWithoutID()
		}
	}

	return newPoll
}

func (q *Question) DeepCopyWithoutID() *Question {
	if q == nil {
		return nil
	}
	newQuestion := &Question{
		Text:   q.Text,
		Type:   q.Type,
		PollID: q.PollID,
		// Explicitly set ID to 0. Copy other gorm.Model fields.
		Model: gorm.Model{
			ID:        0, // ID is reset to 0
			CreatedAt: q.Model.CreatedAt,
			UpdatedAt: q.Model.UpdatedAt,
			DeletedAt: q.Model.DeletedAt,
		},
	}

	// Deep copy the Options slice
	if q.Options != nil {
		newQuestion.Options = make([]Option, len(q.Options))
		for i, opt := range q.Options {
			newQuestion.Options[i] = *opt.DeepCopyWithoutID() // Recursively call DeepCopyWithoutID
		}
	}

	// For the 'Votes' map (which is gorm:"-"):
	// If you want to copy the current votes *data*, you must create a new map
	// and copy its contents. If you want a fresh, empty map for the new object,
	// simply initialize it (as done below). Given it's a "new" object (ID 0),
	// usually you'd want a fresh empty votes map.
	newQuestion.Votes = make(map[string]int)
	// If you wanted to copy the votes data, uncomment these lines:
	// for k, v := range q.Votes {
	// 	newQuestion.Votes[k] = v
	// }

	return newQuestion
}

func (o *Option) DeepCopyWithoutID() *Option {
	if o == nil {
		return nil
	}
	newOption := &Option{
		Text:       o.Text,
		QuestionID: o.QuestionID,
		// Explicitly set ID to 0. Copy other gorm.Model fields.
		Model: gorm.Model{
			ID:        0, // ID is reset to 0
			CreatedAt: o.Model.CreatedAt,
			UpdatedAt: o.Model.UpdatedAt,
			DeletedAt: o.Model.DeletedAt,
		},
	}
	return newOption
}
