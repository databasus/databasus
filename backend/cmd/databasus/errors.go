package main

import "errors"

var (
	ErrCreateUserTakesPassword = errors.New("--create-user takes the password in --password, not --new-password")
	ErrEmailMissing            = errors.New("no email provided, pass one via --email=\"some@email.com\"")
	ErrUnexpectedStatus        = errors.New("unexpected status code")
)
