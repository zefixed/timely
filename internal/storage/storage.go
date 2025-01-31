package storage

import "errors"

var (
    ErrUserNotFound = errors.New("user not found")
    ErrIncorrectAction = errors.New("incorrect action")
)
