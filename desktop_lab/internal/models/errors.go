package models

import (
	"errors"
	"fmt"
)

var (
	ErrReceiving         = errors.New("receiving error")
	ErrFailedToCreate    = errors.New("failed to create")
	ErrFailedToUpdate    = errors.New("failed to update")
	ErrFailedToDelete    = errors.New("failed to delete")
	ErrNotFound          = errors.New("not found")
	ErrNoFieldsToUpdate  = errors.New("no fields to update")
	ErrInvalidUUID       = errors.New("invalid UUID")
	ErrIncorrectPassword = errors.New("incorrect password")
)

func MakeError(dErr, err error, object string) error {
	return fmt.Errorf("%w %v: %w", dErr, object, err)
}
