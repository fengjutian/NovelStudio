package project_test

import (
	"context"
	"errors"
	"testing"

	"novelstudio/internal/project"
)

func TestContentTypeCRUDAndProjectValidation(t *testing.T) {
	ctx := context.Background()
	store := project.NewMemoryStore()
	created, err := store.CreateContentType(ctx, project.CreateContentTypeInput{Code: "MARKETING_COPY", Name: "营销文案", Icon: "营", Accent: "rose", Description: "广告与活动文案", Prompt: "突出品牌价值，不虚构产品参数。"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, project.CreateInput{Name: "新品发布", Type: created.Code}); err != nil {
		t.Fatal(err)
	}
	updated, err := store.UpdateContentType(ctx, created.Code, project.UpdateContentTypeInput{Name: "品牌文案", Icon: "牌", Accent: "violet", Description: "品牌内容", Prompt: "使用统一品牌语气。"})
	if err != nil || updated.Name != "品牌文案" || updated.Prompt != "使用统一品牌语气。" {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	if err := store.DeleteContentType(ctx, created.Code); !errors.Is(err, project.ErrContentTypeInUse) {
		t.Fatalf("expected in-use error, got %v", err)
	}
}

func TestUnusedContentTypeCanBeDeleted(t *testing.T) {
	ctx := context.Background()
	store := project.NewMemoryStore()
	item, err := store.CreateContentType(ctx, project.CreateContentTypeInput{Code: "PODCAST", Name: "播客", Icon: "播"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteContentType(ctx, item.Code); err != nil {
		t.Fatal(err)
	}
	items, _ := store.ListContentTypes(ctx)
	for _, current := range items {
		if current.Code == item.Code {
			t.Fatal("deleted type remains in list")
		}
	}
}
