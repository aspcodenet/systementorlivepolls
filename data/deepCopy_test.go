// go test ./...
// från rooten
// . refers to the current directory (package).

//... is a wildcard that means "and all subdirectories (packages) recursively".

package data

import (
	"testing"

	"gorm.io/gorm"
)

func TestDeepCopyPoll(t *testing.T) {
	// Create a poll with questions and options
	originalPoll := &Poll{
		Title: "Original Poll",
		Questions: []Question{
			{
				Text: "Question 1",
				Options: []Option{
					{Text: "Option 1"},
					{Text: "Option 2"},
				},
			},
			{
				Text: "Question 2",
				Options: []Option{
					{Text: "Option 3"},
					{Text: "Option 4"},
				},
			},
		},
	}

	// Create a deep copy of the poll
	copiedPoll := originalPoll.DeepCopyWithoutID()

	// Check that the copied poll is not the same as the original
	if copiedPoll == originalPoll {
		t.Errorf("Copied poll is the same as the original")
	}

	// Check that the copied poll has the same title
	if copiedPoll.Title != originalPoll.Title {
		t.Errorf("Copied poll title is different from the original")
	}

	// Check that the copied poll has the same questions
	if len(copiedPoll.Questions) != len(originalPoll.Questions) {
		t.Errorf("Copied poll questions are different from the original")
	}

	// Check that the copied poll questions have the same text and options
	for i, question := range copiedPoll.Questions {
		if question.Text != originalPoll.Questions[i].Text {
			t.Errorf("Copied poll question text is different from the original")
		}
		if len(question.Options) != len(originalPoll.Questions[i].Options) {
			t.Errorf("Copied poll question options are different from the original")
		}
		for j, option := range question.Options {
			if option.Text != originalPoll.Questions[i].Options[j].Text {
				t.Errorf("Copied poll question option text is different from the original")
			}
		}
	}
}

func TestDeepCopyPollWithoutID(t *testing.T) {
	// Create a poll with questions and options
	originalPoll := &Poll{
		Model: gorm.Model{ID: 1},
		Title: "Original Poll",
		Questions: []Question{
			{
				Text: "Question 1",
				Options: []Option{
					{Text: "Option 1"},
					{Text: "Option 2"},
				},
			},
			{
				Text: "Question 2",
				Options: []Option{
					{Text: "Option 3"},
					{Text: "Option 4"},
				},
			},
		},
	}

	// Create a deep copy of the poll without ID
	copiedPoll := originalPoll.DeepCopyWithoutID()

	// Check that the copied poll is not the same as the original
	if copiedPoll == originalPoll {
		t.Errorf("Copied poll is the same as the original")
	}

	// Check that the copied poll has no ID
	if copiedPoll.ID != 0 {
		t.Errorf("Copied poll has an ID")
	}

	// Check that the copied poll has the same title
	if copiedPoll.Title != originalPoll.Title {
		t.Errorf("Copied poll title is different from the original")
	}

	// Check that the copied poll has the same questions
	if len(copiedPoll.Questions) != len(originalPoll.Questions) {
		t.Errorf("Copied poll questions are different from the original")
	}

	// Check that the copied poll questions have the same text and options
	for i, question := range copiedPoll.Questions {
		if question.Text != originalPoll.Questions[i].Text {
			t.Errorf("Copied poll question text is different from the original")
		}
		if len(question.Options) != len(originalPoll.Questions[i].Options) {
			t.Errorf("Copied poll question options are different from the original")
		}
		for j, option := range question.Options {
			if option.Text != originalPoll.Questions[i].Options[j].Text {
				t.Errorf("Copied poll question option text is different from the original")
			}
		}
	}
}
