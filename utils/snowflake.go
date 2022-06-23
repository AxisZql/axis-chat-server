package utils

import "github.com/bwmarrin/snowflake"

func GetSnowflakeId() string {
	// default node id eq 1,this can modify to different serverId node
	node, _ := snowflake.NewNode(1)
	// Generate a snowflake ID
	id := node.Generate().String()
	return id
}
