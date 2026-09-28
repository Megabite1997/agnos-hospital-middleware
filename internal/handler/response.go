// Package handler contains the Gin HTTP handlers.
package handler

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Error codes returned in the "error.code" field.
const (
	CodeValidation         = "VALIDATION_ERROR"
	CodeHospitalNotFound   = "HOSPITAL_NOT_FOUND"
	CodeUsernameTaken      = "USERNAME_TAKEN"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeInternal           = "INTERNAL_ERROR"
)

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func respondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": errorBody{Code: code, Message: message}})
}

// respondBindError turns a binding/validation error into a 400 with a
// per-field message map keyed by the JSON field name.
func respondBindError(c *gin.Context, err error, req any) {
	body := errorBody{Code: CodeValidation, Message: "request is invalid"}

	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		body.Fields = make(map[string]string, len(verrs))
		for _, fe := range verrs {
			name := jsonFieldName(req, fe.StructField())
			body.Fields[name] = validationMessage(fe)
		}
	} else {
		body.Message = "request body is malformed"
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": body})
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("must be at least %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("must be at most %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "email":
		return "must be a valid email address"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "alphanum":
		return "must contain only letters and digits"
	default:
		return "is invalid"
	}
}

func jsonFieldName(req any, structField string) string {
	t := reflect.TypeOf(req)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if f, ok := t.FieldByName(structField); ok {
		if tag := strings.Split(f.Tag.Get("json"), ",")[0]; tag != "" && tag != "-" {
			return tag
		}
	}
	return structField
}
