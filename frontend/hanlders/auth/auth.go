package auth

import (
	"bytes"
	"encoding/json"
	"expense-tracker/config"
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/auth"
	"expense-tracker/types"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/register", common.Make(h.handleRegisterGet))
	router.POST("/register", common.Make(h.handleRegisterPost))
	router.POST("/register/validate/email", common.Make(h.handleValidateEmail))
	router.POST("/register/validate/password", common.Make(h.handleValidatePassword))

	router.GET("/login", common.Make(h.handleLoginGet))
	router.POST("/login", common.Make(h.handleLoginPost))
}

func (h *Handler) handleRegisterGet(c *gin.Context) error {
	return common.Render(c.Writer, c.Request, auth.Register())
}

func (h *Handler) handleRegisterPost(c *gin.Context) error {
	apiUrl := "http://" + config.Envs.BackendURL + config.Envs.APIPath

	payload := types.RegisterUserPayload{
		Nickname:  c.PostForm("nickname"),
		Firstname: c.PostForm("firstname"),
		Lastname:  c.PostForm("lastname"),
		Email:     c.PostForm("email"),
		Password:  c.PostForm("password"),
	}
	marshalled, err := json.Marshal(payload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	req, err := http.NewRequest(http.MethodPost, apiUrl+"/register", bytes.NewBuffer(marshalled))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := http.Client{}
	res, err := client.Do(req)
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

	c.Header("HX-Redirect", "/login")
	c.Status(200)

	return nil
}

func (h *Handler) handleValidateEmail(c *gin.Context) error {
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
		emailExist, err := verifyEmail(email)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		if emailExist {
			message += "* this email is used"
		}
	}

	passColor := "text-green-500"
	failColor := "text-red-500"

	html := ""
	if message == "" {
		html = `<div id="email-msg" class="-my-2 text-xs w-full text-center ` +
			passColor + `">Bravo!</div>`
	} else {
		html = `<div id="email-msg" class="-my-2 text-xs w-full text-right ` +
			failColor + `">` + message + "</div>"
	}
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(html))

	return nil
}

func verifyEmail(email string) (bool, error) {
	// payloads
	type emailRequest struct {
		Email string `json:"email"`
	}
	type emailRsp struct {
		Exist bool `json:"exist"`
	}

	// make http request
	payload := emailRequest{
		Email: email,
	}
	apiUrl := "http://" + config.Envs.BackendURL + config.Envs.APIPath

	marshalled, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, apiUrl+"/checkEmail", bytes.NewBuffer(marshalled))
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	// parse response
	rsp := emailRsp{}
	err = json.NewDecoder(res.Body).Decode(&rsp)

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("backend server error")
	}

	return rsp.Exist, nil
}

func (h *Handler) handleValidatePassword(c *gin.Context) error {
	password := c.PostForm("password")
	message := verifyPassword(password)

	passColor := "text-green-500"
	failColor := "text-red-500"

	html := ""
	if message == "" {
		html = `<div id="password-msg" class="-my-2 text-xs w-full text-center ` +
			passColor + `">Bravo!</div>`
	} else {
		html = `<div id="password-msg" class="-my-2 text-xs w-full text-right ` +
			failColor + `">` + message + "</div>"
	}

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(html))

	return nil
}

func verifyPassword(password string) string {
	var uppercasePresent bool
	var lowercasePresent bool
	var numberPresent bool
	var specialCharPresent bool
	const minPassLength = 8
	const maxPassLength = 64
	var passLen int
	var errorString string

	for _, ch := range password {
		switch {
		case unicode.IsNumber(ch):
			numberPresent = true
			passLen++
		case unicode.IsUpper(ch):
			uppercasePresent = true
			passLen++
		case unicode.IsLower(ch):
			lowercasePresent = true
			passLen++
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			specialCharPresent = true
			passLen++
		case ch == ' ':
			passLen++
		}
	}
	appendError := func(err string) {
		if len(strings.TrimSpace(errorString)) != 0 {
			errorString += "<br/>" + err
		} else {
			errorString = err
		}
	}
	if !lowercasePresent {
		appendError("* lowercase letter missing")
	}
	if !uppercasePresent {
		appendError("* uppercase letter missing")
	}
	if !numberPresent {
		appendError("* at least one numeric character required")
	}
	if !specialCharPresent {
		appendError("* special character missing")
	}
	if !(minPassLength <= passLen && passLen <= maxPassLength) {
		appendError(fmt.Sprintf("* password length must be between %d to %d characters long", minPassLength, maxPassLength))
	}

	return errorString
}

func (h *Handler) handleLoginGet(c *gin.Context) error {
	return common.Render(c.Writer, c.Request, auth.Login())
}

func (h *Handler) handleLoginPost(c *gin.Context) error {
	apiUrl := "http://" + config.Envs.BackendURL + config.Envs.APIPath

	payload := types.LoginUserPayload{
		Email:    c.PostForm("email"),
		Password: c.PostForm("password"),
	}
	marshalled, err := json.Marshal(payload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	req, err := http.NewRequest(
		http.MethodPost, apiUrl+"/login", bytes.NewBuffer(marshalled))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		resErr := types.ServerErr{}
		err = json.NewDecoder(res.Body).Decode(&resErr)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		c.Writer.Write([]byte(resErr.Error))
		c.Status(http.StatusInternalServerError)
		return fmt.Errorf(resErr.Error)
	}

	token := types.LoginResponse{}
	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	c.Header("HX-Redirect", "/")
	c.SetCookie(
		"access_token", token.Token,
		int(config.Envs.JWTExpirationInSeconds),
		"/", "localhost", false, true)
	c.Status(200)

	return nil
}
