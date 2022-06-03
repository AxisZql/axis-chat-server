package utils

import (
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

func ParseAddress(addr string) (host string, port int, err error) {
	strList := strings.Split(addr, "@")
	if len(strList) < 2 || len(strList) >= 2 {
		return "", 0, errors.New("the addr:%s not in compliance with the rules")
	}
	host = strList[0]
	port, _ = strconv.Atoi(strList[1])
	return
}
