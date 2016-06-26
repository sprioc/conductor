package handlers

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"gopkg.in/mgo.v2/bson"

	"github.com/gorilla/mux"
	"github.com/sprioc/sprioc-core/pkg/authentication"
	"github.com/sprioc/sprioc-core/pkg/authorization"
	"github.com/sprioc/sprioc-core/pkg/contentStorage"
	"github.com/sprioc/sprioc-core/pkg/model"
	"github.com/sprioc/sprioc-core/pkg/store"
)

type signUpFields struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func Signup(w http.ResponseWriter, r *http.Request) Response {
	decoder := json.NewDecoder(r.Body)

	newUser := signUpFields{}

	err := decoder.Decode(&newUser)
	if err != nil {
		return Resp("Bad Request", http.StatusBadRequest)
	}

	if mongo.ExistsUserName(newUser.Username) || mongo.ExistsEmail(newUser.Email) {
		return Resp("Username or Email already exist", http.StatusConflict)
	}

	password, salt, err := authentication.GetSaltPass(newUser.Password)
	if err != nil {
		return Resp("Error adding user", http.StatusConflict)
	}

	usr := model.User{
		ID:        bson.NewObjectId(),
		Email:     newUser.Email,
		Pass:      password,
		Salt:      salt,
		ShortCode: newUser.Username,
	}

	err = store.CreateUser(mongo, usr)
	if err != nil {
		return Resp("Error adding user", http.StatusConflict)
	}

	return Response{Code: http.StatusAccepted}
}

func AvatarUpload(w http.ResponseWriter, r *http.Request) Response {
	user, userRef, err := getLoggedInUser(r)
	if err != nil {
		return err.(Response)
	}

	file, err := ioutil.ReadAll(r.Body)
	n := len(file)

	if n == 0 {
		return Resp("Cannot upload file with 0 bytes.", http.StatusBadRequest)
	}

	err = contentStorage.ProccessImage(file, n, user.ShortCode, "avatar")
	if err != nil {
		log.Println(err)
		return Resp(err.Error(), http.StatusBadRequest)
	}

	sources := formatAvatarSources(user.ShortCode)

	err = store.ModifyAvatar(mongo, userRef, sources)
	if err != nil {
		return Resp("Unable to add image", http.StatusInternalServerError)
	}
	return Response{Code: http.StatusAccepted}
}

func formatAvatarSources(shortcode string) model.ImgSource {
	const prefix = "https://images.sprioc.xyz/avatars/"
	var resourceBaseURL = prefix + shortcode
	return model.ImgSource{
		Raw:    resourceBaseURL,
		Large:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy",
		Medium: resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=1080&fit=max",
		Small:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=400&fit=max",
		Thumb:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=200&fit=max",
	}
}

func GetUser(w http.ResponseWriter, r *http.Request) Response {
	UID := mux.Vars(r)["username"]

	user, err := store.GetByUserName(mongo, UID)
	if err != nil {
		return Resp("Not Found", http.StatusNotFound)
	}

	dat, err := json.Marshal(user)
	if err != nil {
		return Resp("Unable to write JSON", http.StatusInternalServerError)
	}

	return Response{Code: http.StatusOK, Data: dat}
}

func DeleteUser(w http.ResponseWriter, r *http.Request) Response {
	user, userRef, err := getUser(r)
	if err != nil {
		return err.(Response)
	}

	loggedUser, _, err := getLoggedInUser(r)
	if err != nil {
		return err.(Response)
	}

	_, err = authorization.Authorized(loggedUser, user)
	if err != nil {
		return Resp(err.Error(), http.StatusUnauthorized)
	}

	err = store.DeleteUser(mongo, userRef)
	if err != nil {
		return Resp("Internal Server Error", http.StatusInternalServerError)
	}
	return Response{Code: http.StatusAccepted}
}

func FavoriteUser(w http.ResponseWriter, r *http.Request) Response {
	user, userRef, err := getUser(r)
	if err != nil {
		return err.(Response)
	}

	loggedInUser, loggedInRef, err := getLoggedInUser(r)
	if err != nil {
		return err.(Response)
	}

	_, err = authorization.Authorized(loggedInUser, user)
	if err != nil {
		return Resp(err.Error(), http.StatusUnauthorized)
	}

	err = store.FavoriteImage(mongo, loggedInRef, userRef)
	if err != nil {
		return Resp("Internal Server Error", http.StatusInternalServerError)
	}
	return Response{Code: http.StatusAccepted}
}

func FollowUser(w http.ResponseWriter, r *http.Request) Response {
	user, userRef, err := getUser(r)
	if err != nil {
		return err.(Response)
	}

	loggedInUser, loggedInRef, err := getLoggedInUser(r)
	if err != nil {
		return err.(Response)
	}

	_, err = authorization.Authorized(loggedInUser, user)
	if err != nil {
		return Resp(err.Error(), http.StatusUnauthorized)
	}

	err = store.FollowUser(mongo, loggedInRef, userRef)
	if err != nil {
		return Resp("Internal Server Error", http.StatusInternalServerError)
	}
	return Response{Code: http.StatusAccepted}
}
