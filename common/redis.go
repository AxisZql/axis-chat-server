package common

/**
*Author: AxisZql
*Date: 2022-5-31
*DESC: redis相关配置
 */

type Filed string

// SESSION map token to userinfo
const SESSION Filed = "axis:session:%s"

const USERID_MAP_TOKEN Filed = "axis:user_map_token:%d"

// GROUP_ONLINE_USER map groupId to record the group all online user`s id
const GROUP_ONLINE_USER Filed = "axis:group_online_user:%d"

// GROUP_ONLINE_USER_COUNT map groupId to the group online user count
const GROUP_ONLINE_USER_COUNT = "axis:group_online_user_count:%d"
