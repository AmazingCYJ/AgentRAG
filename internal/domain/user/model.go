package user

// User 表示系统中的当前最小用户模型。
type User struct {
	UserID   string
	Username string
	Password string
	Role     string
	Avatar   string
}

// Profile 表示返回给前端的用户资料。
type Profile struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Avatar   string `json:"avatar,omitempty"`
}

// ToProfile 将用户转换为前端可消费的资料结构。
func (u User) ToProfile() Profile {
	return Profile{
		UserID:   u.UserID,
		Username: u.Username,
		Role:     u.Role,
		Avatar:   u.Avatar,
	}
}
