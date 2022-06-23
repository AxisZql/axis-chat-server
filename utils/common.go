package utils

import (
	"axisChat/utils/zlog"
	"crypto/rand"
	"encoding/base64"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"io"
	"strconv"
	"strings"
)

func ParseAddress(addr string) (host string, port int, err error) {
	strList := strings.Split(addr, "@")
	if len(strList) < 2 || len(strList) > 2 {
		return "", 0, errors.New("the addr:%s not in compliance with the rules")
	}
	host = strList[0]
	port, _ = strconv.Atoi(strList[1])
	return
}

func EncPassword(pwd string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	if err != nil {
		zlog.Error(err.Error())
		return ""
	}
	return string(hash)
}

func VerifyPwd(hash, pwd string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
	if err != nil {
		zlog.Debug(err.Error())
		return false
	}
	return true
}

// GetRandomToken 生成随机登陆凭证（token）
func GetRandomToken(length int) string {
	r := make([]byte, length)
	_, _ = io.ReadFull(rand.Reader, r)
	return base64.URLEncoding.EncodeToString(r)
}
