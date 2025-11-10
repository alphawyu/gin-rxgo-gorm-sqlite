package util

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var (
	LOGIN_ERROR = errors.New("Not Registered email or invalid password")
)

type CommonError struct {
	Errors map[string]interface{} `json:"errors"`
}

func NewError(key string, err error) CommonError {
	res := CommonError{}
	res.Errors = make(map[string]interface{})
	res.Errors[key] = err.Error()
	return res
}

func NewValidatorError(err error) CommonError {
	res := CommonError{}
	res.Errors = make(map[string]interface{})
	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, v := range errs {
			if v.Param() != "" {
				res.Errors[v.Field()] = fmt.Sprintf("{%s: %s}", v.Tag(), v.Param())
			} else {
				res.Errors[v.Field()] = fmt.Sprintf("{key: %s}", v.Tag())
			}

		}
	} else {
		res.Errors["error"] = err.Error()
	}
	return res
}
