package utils

import (
	"fmt"
	"testing"
)

func TestHashBalance(t *testing.T) {
	h := NewHashBalance(16, nil)
	h.Add("uii")
	h.Add("ioo0")
	// 同一个key，选择的节点永远相同
	ans, _ := h.Get("hello")
	fmt.Println(ans)
}
