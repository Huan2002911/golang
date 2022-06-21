package user

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"golang_swagger/orm"
	"net/http"
)

// @Tags user
// @Summary 用户登录
// @Description
// @Accept  application/json
// @Produce application/json
// @Param data body orm.Login true "请求体"
// @Success 200 {object} orm.LoginResponse "成功"
// @Router /swagger/user/login [post]
func Login(c *gin.Context) {
	w := c.Writer

	var login = orm.Login{}

	if err := json.NewDecoder(c.Request.Body).Decode(&login); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	response := orm.LoginResponse{}
	if login.User == "lion" && login.PassWord == "123 " {
		response.Message = "Success"
		response.Code = 1001
	} else {
		response.Message = "failure"
		response.Code = 1002
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

}
