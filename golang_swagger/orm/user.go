package orm

// Login Login结构
type Login struct {
	User     string `json:"user,omitempty"`
	PassWord string `json:"passWord,omitempty"`
}

//Login Response 响应结构
type LoginResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
