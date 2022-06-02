package logic

import (
	"axisChat/logic/proto"
	"context"
	"github.com/golang/protobuf/ptypes/empty"
)

/**
*Author:AxisZql
*Date:2022-5-31
*DESC:implement the rpc server interface
 */

type Server struct{}

// Connect 当客户端通过connect层连接服务时，调用该rpc方法通过查看redis中有没有该用户的登陆信息，从而达到鉴权的目的，如果是
// 私聊的话将该用户的上线信息写入其好友的所有MQ中，如果是群聊的话，将该群的人数信息推送到群聊对应的MQ中
func (s *Server) Connect(ctx context.Context, request *proto.ConnectRequest) (*proto.ConnectReply, error) {

	return &proto.ConnectReply{}, nil
}

// DisConnect 当connect层检测到对应客户端下线后，更新redis中该用户的在线信息，并在好友和其所在群聊的MQ中写入其下线的信息
func (s *Server) DisConnect(ctx context.Context, request *proto.DisConnectRequest) (*empty.Empty, error) {
	return nil, nil
}

func (s *Server) Register(ctx context.Context, request *proto.RegisterRequest) (*proto.RegisterReply, error) {
	return nil, nil
}

func (s *Server) Login(ctx context.Context, request *proto.LoginRequest) (*proto.LoginReply, error) {
	return nil, nil
}

func (s *Server) LoginOut(ctx context.Context, request *proto.LoginOutRequest) (*empty.Empty, error) {
	return nil, nil
}

func (s *Server) GetUserInfoByAccessToken(ctx context.Context, request *proto.GetUserInfoByAccessTokenRequest) (*proto.GetUserInfoByAccessTokenReply, error) {
	return nil, nil
}

func (s *Server) GetUserInfoByUserid(ctx context.Context, request *proto.GetUserInfoByUseridRequest) (*proto.GetUserInfoByUseridReply, error) {
	return nil, nil
}

func (s *Server) UpdateUserInfo(ctx context.Context, request *proto.UpdateUserInfoRequest) (*proto.UpdateUserInfoReply, error) {
	return nil, nil
}

func (s *Server) UpdatePassword(ctx context.Context, request *proto.UpdatePasswordRequest) (*empty.Empty, error) {
	return nil, nil
}

func (s *Server) SearchUser(ctx context.Context, request *proto.SearchUserRequest) (*proto.SearchUserReply, error) {
	return nil, nil
}

func (s *Server) SearchGroup(ctx context.Context, request *proto.SearchGroupRequest) (*proto.SearchGroupReply, error) {
	return nil, nil
}

func (s *Server) GetGroupMsgByPage(ctx context.Context, request *proto.GetGroupMsgByPageRequest) (*proto.GetGroupMsgByPageReply, error) {
	return nil, nil
}

func (s *Server) GetFriendMsgByPage(ctx context.Context, request *proto.GetFriendMsgByPageRequest) (*proto.GetFriendMsgByPageReply, error) {
	return nil, nil
}

func (s *Server) CreateGroup(ctx context.Context, group *proto.Group) (*proto.Group, error) {
	return nil, nil
}

func (s *Server) AddGroup(ctx context.Context, request *proto.AddGroupRequest) (*empty.Empty, error) {
	return nil, nil
}

func (s *Server) AddFriend(ctx context.Context, request *proto.AddFriendRequest) (*empty.Empty, error) {
	return nil, nil
}

func (s *Server) Push(ctx context.Context, request *proto.PushRequest) (*empty.Empty, error) {
	return nil, nil
}

func (s *Server) PushRoom(ctx context.Context, request *proto.PushRoomRequest) (*empty.Empty, error) {
	return nil, nil
}

func (s *Server) PushRoomCount(ctx context.Context, request *proto.PushRoomCountRequest) (*empty.Empty, error) {
	return nil, nil
}

func (s *Server) PushRoomInfo(ctx context.Context, request *proto.PushRoomInfoRequest) (*empty.Empty, error) {
	return nil, nil
}
