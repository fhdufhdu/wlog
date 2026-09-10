package post

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateDraftAndPublication(t *testing.T) {
	empty := Form{}
	if _, err := validate(empty, false); err != nil {
		t.Fatalf("draft rejected: %v", err)
	}
	if _, err := validate(empty, true); err == nil {
		t.Fatal("incomplete publication accepted")
	}
	form := Form{Title: " 제목 ", Slug: "Hello World", TopicID: uuid.NewString(), ContentMarkdown: "**자동 설명**"}
	got, err := validate(form, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "제목" || got.Slug != "hello-world" || got.Description != "자동 설명" {
		t.Fatalf("unexpected validated post: %#v", got)
	}
}
