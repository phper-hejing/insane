package utils

import (
	"strconv"
)

func ConvString(i interface{}) (s string) {
	switch i.(type) {
	case int:
		s = strconv.Itoa(i.(int))
	case uint64:
		s = strconv.FormatUint(i.(uint64), 10)
	case int64:
		s = strconv.FormatInt(i.(int64), 10)
	case float64:
		s = strconv.FormatFloat(i.(float64), 'f', -1, 64)
	case string:
		s = i.(string)
	default:
		s = ""
	}
	return
}
