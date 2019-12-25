package utils

import (
	"fmt"
	"math/rand"
	"strconv"
)

func GetRandomIntRange(num int) uint64 {
	return uint64(rand.Intn(num))
}

func GetRandomStrings(len int64) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rs := ""
	var i int64
	for i = 0; i < len; i++ {
		r := rand.Intn(62)
		if r == 0 {
			r = 1
		}
		rs += str[r-1 : r]
	}
	return rs
}
func GetRandomintegers(len int64) int64 {
	var (
		in string
		i  int64
	)
	for i = 0; i < len; i++ {
		in += fmt.Sprintf("%d", rand.Intn(10))
	}
	n, _ := strconv.ParseInt(in, 10, 64)
	return n
}
