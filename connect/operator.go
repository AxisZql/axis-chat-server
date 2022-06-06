package connect

type Operator interface {
	Connect(accessToken, serverId string) (userid int64, err error)
	DisConnect(userid int64) (err error)
}

type DefaultOperator struct{}

// Connect rpc call logic layer
func (o *DefaultOperator) Connect(accessToken, serverId string) (userid int64, err error) {
	rpcConnect := new(Connect)
	userid, err = rpcConnect.Connect(accessToken, serverId)
	return
}

// DisConnect rpc call logic layer
func (o *DefaultOperator) DisConnect(userid int64) (err error) {
	rpcConnect := new(Connect)
	err = rpcConnect.DisConnect(userid)
	return
}
