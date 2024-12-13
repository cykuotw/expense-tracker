package group

import (
	"encoding/json"
	"expense-tracker/frontend/hanlders/auth"
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/index"
	"expense-tracker/types"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/create_group", common.Make(h.handleCreateNewGroupGet))
	router.POST("/create_group", common.Make(h.handleCreateNewGroup))

	router.GET("/groups", common.Make(h.handleGetGroupList))
	router.GET("/group/:groupId", common.Make(h.handleGetGroup))

	router.GET("/groupSelect/:groupId", common.Make(h.handleGetGroupSelect))

	router.GET("/add_member", common.Make(h.handleGetAddNewMember))
	router.POST("/add_member", common.MakeMuitiErr(h.handlePostAddNewMember))
	router.POST("/add_member_list", common.Make(h.handlePostAddNewMemberList))
	router.POST("/check_member_exist", common.Make(h.handleCheckMemberExist))
}

func (h *Handler) handleCreateNewGroupGet(c *gin.Context) error {
	return common.Render(c.Writer, c.Request, index.NewGroup())
}

func (h *Handler) handleCreateNewGroup(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	payload := types.CreateGroupPayload{
		GroupName:   c.PostForm("groupname"),
		Description: c.PostForm("description"),
		Currency:    c.PostForm("currency"),
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodPost, "/create_group", token, payload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		resErr := types.ServerErr{}
		err = json.NewDecoder(res.Body).Decode(&resErr)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		c.Status(http.StatusInternalServerError)
		return fmt.Errorf(resErr.Error)
	}

	c.Header("HX-Redirect", "/")
	c.Status(200)

	return nil
}

func (h *Handler) handleGetGroupList(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/groups", token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	payloadList := []types.GetGroupListResponse{}
	err = json.NewDecoder(res.Body).Decode(&payloadList)

	html := ""
	for _, payload := range payloadList {
		html += `
			<div class="card w-full lg:w-1/5 md:w-1/3 bg-base-100 shadow-md m-2 mx-6 md:m-2">
				<a href="/group/` +
			payload.ID +
			`">
					<div class="card-body">
						<div class="card-title">` +
			payload.GroupName +
			`</div>
					<p class="break-all">` +
			payload.Description +
			`</p>
					</div>
				</a>
			</div>
		`
	}

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(html))

	return nil
}

func (h *Handler) handleGetGroup(c *gin.Context) error {
	// get group details
	groupId := c.Param("groupId")

	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	resGroup, err := common.MakeBackendHTTPRequest(http.MethodGet, "/group/"+groupId, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer resGroup.Body.Close()

	payloadGroup := types.GetGroupResponse{}
	err = json.NewDecoder(resGroup.Body).Decode(&payloadGroup)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	// get balance
	resBalance, err := common.MakeBackendHTTPRequest(http.MethodGet, "/balance/"+groupId, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer resBalance.Body.Close()

	payloadBalance := types.BalanceResponse{}
	err = json.NewDecoder(resBalance.Body).Decode(&payloadBalance)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	// get expenses list
	resExpenseList, err := common.MakeBackendHTTPRequest(http.MethodGet, "/expense_list/"+groupId, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer resExpenseList.Body.Close()

	payloadExpenseList := []types.ExpenseResponseBrief{}
	if resExpenseList.StatusCode == http.StatusOK {
		err = json.NewDecoder(resExpenseList.Body).Decode(&payloadExpenseList)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
	}

	return common.Render(c.Writer, c.Request,
		index.GroupDetail(groupId, payloadGroup.GroupName, payloadBalance, payloadExpenseList))
}

func (h *Handler) handleGetGroupSelect(c *gin.Context) error {
	groupId := c.Param("groupId")

	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/groups", token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	payloadList := []types.GetGroupListResponse{}
	if res.StatusCode == http.StatusOK {
		err = json.NewDecoder(res.Body).Decode(&payloadList)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
	}

	html := `<select class="select select-bordered w-full text-base text-center" id="groupId" name="groupId">`
	for _, payload := range payloadList {
		if payload.ID == groupId {
			html += "<option value=\"" + payload.ID + "\" selected>"
		} else {
			html += "<option value=\"" + payload.ID + "\">"
		}
		html += payload.GroupName + "</option>"
	}
	html += "</select>"

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(html))

	return nil
}

func (h *Handler) handleGetAddNewMember(c *gin.Context) error {
	groupId := c.Query("g")

	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}
	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/related_member?g="+groupId, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	relatedUserList := []types.RelatedMember{}
	if res.StatusCode == http.StatusOK {
		err = json.NewDecoder(res.Body).Decode(&relatedUserList)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
	}

	return common.Render(c.Writer, c.Request, index.AddMember(groupId, relatedUserList))
}

func (h *Handler) handleCheckMemberExist(c *gin.Context) error {
	email := c.PostForm("email")

	matched, err := regexp.MatchString("^[A-Za-z0-9._%+\\-]+@[A-Za-z0-9\\-]+\\.[a-z]{2,4}$", email)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	message := ""
	if !matched {
		message += "* invalid email format (example@youremail.com)"
	} else {
		emailExist, err := auth.VerifyEmail(email)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		if !emailExist {
			message += "* user not exist"
		}
	}

	html := `<div class="text-xs w-full text-center `
	if message == "" {
		html += `text-green-500"> 
				<button
					class="btn btn-sm btn-outline bg-base-100 text-green-500 font-normal"
					hx-post="/add_member_list"
					hx-target="#members"
					hx-swap="beforeend"
					>add to list?</button></div>`
	} else {
		html += `text-red-500">` + message + "</div>"
	}

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(html))

	return nil
}

func (h *Handler) handlePostAddNewMemberList(c *gin.Context) error {
	email := c.PostForm("email")

	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	// payloads
	type emailRequest struct {
		Email string `json:"email"`
	}

	// make http request
	payload := emailRequest{
		Email: email,
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/userInfo", token, payload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	user := types.User{}
	if res.StatusCode == http.StatusOK {
		err = json.NewDecoder(res.Body).Decode(&user)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
	}
	html := `<label class="label cursor-pointer">
				<span class="label-text">` + user.Username + `</span>
				<input type="checkbox" is="boolean-checkbox" checked class="checkbox" name="candidate[]" value=` + user.ID.String() + `>
			</label>`

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(html))

	return nil
}

func (h *Handler) handlePostAddNewMember(c *gin.Context) []error {
	groupId := c.Query("g")

	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return []error{err}
	}

	candidates := c.PostFormArray("candidate[]")
	var errs []error
	for _, candidate := range candidates {
		go func() {
			var jsonMap map[string]interface{}
			json.Unmarshal([]byte(candidate), &jsonMap)

			payload := types.UpdateGroupMemberPayload{
				Action:  "add",
				GroupID: groupId,
				UserID:  jsonMap["value"].(string),
			}
			if !jsonMap["checked"].(bool) {
				payload.Action = "delete"
			}

			res, err := common.MakeBackendHTTPRequest(http.MethodPut, "/group_member", token, payload)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				errs = append(errs, err)
			}
			defer res.Body.Close()
		}()
	}

	if len(errs) != 0 {
		c.Header("HX-Redirect", "/add_member?g="+groupId)
		c.Status(200)
		return errs
	}

	c.Header("HX-Redirect", "/group/"+groupId)
	c.Status(200)

	return nil
}
