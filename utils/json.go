package utils

import (
	"errors"
	"github.com/tidwall/gjson"
)

func ParseJson(json string) (result gjson.Result, err error) {
	if !gjson.Valid(json) {
		err = errors.New("invalid json")
		return
	}
	result = gjson.Parse(json)
	return
}
