package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	modelsAuth "passer-auth-service-v4/pkg/models/auth"
	"passer-auth-service-v4/pkg/utils"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// AuthCtl is a struct that represents a authentication controller, in a MVC pattern.
type AuthCtl struct {
	db   *sql.DB
	name string
}

var (
	ErrAuthFail                error = errors.New("[API-Users]: authentication failure")
	ErrNotAllowedRequestMethod error = errors.New("[API-Users]: requst method is not allowed for this endpoint")
	ErrUserExisted             error = errors.New("[API-Users]: user already existed")
	ErrEnvNotLoaded                  = errors.New("[JWT]: fail to load the env file")
	ErrPayloadParsing                = errors.New("[JWT]: fail to parse payload")
)

// NewAuthCtl sets:
// - the database connection pools to use
// - the name assigned to this struct (for reference purpoe)
func NewAuthCtl(db *sql.DB, name string) *AuthCtl {
	return &AuthCtl{db: db, name: name}
}

// Auth executes the authentication flow using:
// - an email
// - password
// submitted in the JSON body of the POST request.
func (a *AuthCtl) Auth(w http.ResponseWriter, r *http.Request) {

	// set the response header
	w.Header().Set("Content-Type", "application/json")

	// get .env values
	err := godotenv.Load("../../.env")
	if err != nil {
		customErr := errors.New(`[AUTH-CTL] fail to load .env parameters`)
		utils.SendErrorMsgToClient(&w, customErr)
		return
		//
	}

	// set the jwt issuer value
	JWT_ISSUER := os.Getenv("JWT_ISSUER")

	// set the jwt expiry time lapse (in minutes)
	JWT_EXP_MINUTES, err := strconv.Atoi(os.Getenv("JWT_EXP_MINUTES"))
	if err != nil {
		customErr := errors.New(`[AUTH-CTL] fail to load .env parameters`)
		utils.SendErrorMsgToClient(&w, customErr)
		return
	}

	// get the JSON body
	authInput := modelsAuth.AuthParams{}
	am := modelsAuth.New(a.db)

	err = utils.ParseBody(r, &authInput)
	if err != nil {
		customErr := errors.New(`[AUTH-CTL] fail to parse JSON body`)
		utils.SendErrorMsgToClient(&w, customErr)
		return
	}

	// execute the authentication operation.
	found, err := am.Auth(authInput)
	if err != nil {
		customErr := errors.New(`[AUTH-CTL] fail to authenticate`)
		utils.SendForbiddenMsgToClient(&w, customErr)
		return
	}

	// ok. authentication passed.
	// generate JWT.
	exp := time.Now().Add(time.Minute * time.Duration(JWT_EXP_MINUTES)).UnixMilli()
	pl := utils.JWTPayload{
		Id:        found.Email,
		NameFirst: found.NameFirst,
		NameLast:  found.NameLast,
		IsAgent:   found.IsAgent,
		IsActive:  found.IsActive,
		Iss:       JWT_ISSUER,
		Exp:       exp,
	}

	var token string
	token, err = generateJWT(pl)
	if err != nil {
		customErr := errors.New(`[AUTH-CTL] fail to generate JWT`)
		utils.SendForbiddenMsgToClient(&w, customErr)
		return
	}

	// set the Authorization header attribute in the response.
	msg := fmt.Sprintf(`{
		"ok" : true,
		"msg" : "[AUTH-CTL]: authentication ok",
		"data" : {
			"token" : "%s",
			"nameFirst" : "%s",
			"nameLast" : "%s",
			"email" : "%s"
		}
	}`, token, found.NameFirst, found.NameLast, found.Email)

	bearerToken := fmt.Sprintf("Bearer %s", token)
	w.Header().Set("Authorization", bearerToken)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(msg))
	//
}

func (a *AuthCtl) VerifyToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	body := `{
		"ok" : true,
		"msg" : "Reached VerifyToken endpoint.",
		"data" : {}
	}`

	w.Write([]byte(body))
}

// generateJWT will generate a JWT using the header and payload passed in.
func generateJWT(payload utils.JWTPayload) (string, error) {

	// get .env values
	err := godotenv.Load("../../.env")
	if err != nil {
		return "", ErrEnvNotLoaded
	}
	JWT_SECRET_KEY := os.Getenv("JWT_SECRET_KEY")

	header := `{
		"alg": "SHA512",
		"typ" : "JWT"
	}`

	// convert payload data to json string
	pl, err := json.Marshal(payload)
	if err != nil {
		return "", ErrPayloadParsing
	}

	token := utils.Generate(header, string(pl), JWT_SECRET_KEY)

	return token, nil
}
