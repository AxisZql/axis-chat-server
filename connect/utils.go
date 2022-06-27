package connect

import (
	"axisChat/common"
	"encoding/json"
)

// DealWebSocketResp 统一处理WebSocket响应
func DealWebSocketResp(msg []byte) []byte {
	var msgSend common.MsgSend
	resp := make([]string, 0)

	_ = json.Unmarshal(msg, &msgSend)
	switch msgSend.Op {
	case common.OpGroupMsgSend:
		resp = append(resp, "groupMsg", string(msg))
		data, _ := json.Marshal(&resp)
		return data
	case common.OpGroupInfoSend:
		resp = append(resp, "groupInfo", string(msg))
		data, _ := json.Marshal(&resp)
		return data
	case common.OpGroupOlineUserCountSend:
		resp = append(resp, "groupCount", string(msg))
		data, _ := json.Marshal(&resp)
		return data
	case common.OPFriendOffOnlineSend:
		resp = append(resp, "friendOff", string(msg))
		data, _ := json.Marshal(&resp)
		return data
	case common.OpFriendOnlineSend:
		resp = append(resp, "friendOn", string(msg))
		data, _ := json.Marshal(&resp)
		return data
	case common.OpFriendMsgSend:
		resp = append(resp, "friendMsg", string(msg))
		data, _ := json.Marshal(&resp)
		return data
	}
	return []byte{}
}
