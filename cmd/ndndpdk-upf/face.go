package main

import (
	"context"

	"go.uber.org/zap"
)

func createFace(ctx context.Context, loc any) (id string, e error) {
	var reply struct {
		ID string `json:"id"`
	}
	e = client.Do(ctx, `
		mutation createFace($locator: JSON!) {
			createFace(locator: $locator) {
				id
			}
		}
	`, map[string]any{
		"locator": loc,
	}, "createFace", &reply)
	if e != nil {
		return "", e
	}
	logger.Info("face created", zap.Any("locator", loc), zap.String("face-id", reply.ID))
	return reply.ID, nil
}

func destroyFace(ctx context.Context, id string) error {
	deleted, e := client.Delete(ctx, id)
	if e != nil {
		return e
	}
	logger.Info("face destroyed", zap.Bool("deleted", deleted), zap.String("face-id", id))
	return nil
}
