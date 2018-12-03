package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	libDatabox "github.com/me-box/lib-go-databox"
)

func index(w http.ResponseWriter, r *http.Request) {

	callbackUrl := r.FormValue("post_auth_callback")
	PostAuthCallbackUrl := "/core-ui/ui/view/" + BasePath + "/info"
	if DataboxTestMode {
		PostAuthCallbackUrl = "/ui/info"
	}
	if callbackUrl != "" {
		PostAuthCallbackUrl = callbackUrl
	}

	if userAuthenticated && callbackUrl != "" {
		//use the callbackUrl if we are logged in and we have one
		if DataboxTestMode {
			fmt.Fprintf(w, "<html><head><script>window.location = '%s';</script><head><body><body></html>", PostAuthCallbackUrl)
		} else {
			fmt.Fprintf(w, "<html><head><script>window.parent.location = '%s';</script><head><body><body></html>", PostAuthCallbackUrl)
		}
		return
	}

	if userAuthenticated {
		http.Redirect(w, r, Host+"/ui/info", 302)
		return
	}

	body := `<!doctype html>
	<head>
	  <meta charset="utf-8">
	  <meta http-equiv="x-ua-compatible" content="ie=edge">
	  <title></title>
	  <meta name="description" content="">
	  <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">

	  <link rel="stylesheet" href="` + BasePath + `/ui/css/normalize.css">
	  <link rel="stylesheet" href="` + BasePath + `/ui/css/main.css">
	  <link rel="stylesheet" href="` + BasePath + `/ui/css/insta.css">
	</head>

	<body>
	  	<div class="form-login">
			<img class="logo" src="` + BasePath + `/ui/img/Instagram_logo.svg" />
			<p>Sign in with your instagram account to download your photos.</p>
			<form action="` + BasePath + `/ui/auth" method="post">
				<div class="row"> <label for="username">Username </label><input autocomplete="off" type="text" name="username" required></div>
				<div class="row"> <label for="password">Password </label><input autocomplete="off" type="password" name="password" required></div>
				<div class="row"> <input type="submit" class="btn-login" value="Sign in"></div>
				<input style="display: none" type="text" name="post_auth_callback" value="` + PostAuthCallbackUrl + `"/>
			</form>
		</div>
	</body>
	</html>`

	fmt.Fprintf(w, body)

}

func info(w http.ResponseWriter, r *http.Request) {

	photosJSON, err := storeClient.KVJSON.Read("InstagramDatastore", "meta")
	libDatabox.ChkErr(err)

	var photos []IMAGE
	json.Unmarshal(photosJSON, &photos)

	photosHTML := ""
	for _, p := range photos {
		photosHTML += `<img class="insta-img" src="` + p.DispURL + `" />`
	}
	if photosHTML == "" {
		photosHTML = "<center><h1>Downloading images</h1><br/><h2>Please come back later ..... </h2></center>"
	}

	body := `<!doctype html>
	<html class="no-js" lang="">

	<head>
	<meta charset="utf-8">
	<meta http-equiv="x-ua-compatible" content="ie=edge">
	<title></title>
	<meta name="description" content="">
	<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">

	<link rel="stylesheet" href="` + BasePath + `/ui/css/normalize.css">
	<link rel="stylesheet" href="` + BasePath + `/ui/css/main.css">
	<link rel="stylesheet" href="` + BasePath + `/ui/css/insta.css">
	</head>

	<body>
	<img class="logo" src="` + BasePath + `/ui/img/Instagram_logo.svg" />
	<div style="float:right"><a href="` + BasePath + `/ui/logout">logout</a></div>
	<div style="clear: both;">%s</div>
	</body>
	</html>`

	fmt.Fprintf(w, body, photosHTML)

}

func logout(w http.ResponseWriter, r *http.Request) {
	err := storeClient.KVText.Delete("InstagramCred", "username")
	if err != nil {
		libDatabox.Err("Error Deleting Datasource " + err.Error())
	}
	err = storeClient.KVText.Delete("InstagramCred", "password")
	if err != nil {
		libDatabox.Err("Error Deleting Datasource " + err.Error())
	}
	err = storeClient.KVJSON.Delete("InstagramDatastore", "meta")
	if err != nil {
		libDatabox.Err("Error Deleting Datasource " + err.Error())
	}
	userAuthenticated = false
	if isRuning {
		close(StopDoDriverWork)
	}
	http.Redirect(w, r, Host+"/ui", 302)
}

func login(w http.ResponseWriter, r *http.Request) {
	libDatabox.Info("Obtained auth")

	r.ParseForm()
	//Obtain user login details for their youtube account
	for k, v := range r.Form {
		if k == "username" {
			err := storeClient.KVText.Write("InstagramCred", "username", []byte(strings.Join(v, "")))
			if err != nil {
				libDatabox.Err("Error Write Datasource " + err.Error())
			}

		} else if k == "password" {
			err := storeClient.KVText.Write("InstagramCred", "password", []byte(strings.Join(v, "")))
			if err != nil {
				libDatabox.Err("Error Write Datasource " + err.Error())
			}
		} else {
			fmt.Println("Finished Storing")
		}

	}
	channel := make(chan bool)
	go infoCheck(channel)
	checkOK := <-channel
	if checkOK {
		userAuthenticated = true
		if !isRuning {
			StopDoDriverWork = make(chan struct{})
			go doDriverWork(StopDoDriverWork)
		}

		callbackUrl := r.FormValue("post_auth_callback")
		PostAuthCallbackUrl := "/core-ui/ui/view/" + BasePath + "/info"
		if callbackUrl != "" {
			PostAuthCallbackUrl = callbackUrl
		}

		if DataboxTestMode {
			fmt.Fprintf(w, "<html><head><script>window.location = '%s';</script><head><body><body></html>", PostAuthCallbackUrl)
		} else {
			fmt.Fprintf(w, "<html><head><script>window.parent.location = '%s';</script><head><body><body></html>", PostAuthCallbackUrl)
		}

	} else {
		userAuthenticated = false
		fmt.Fprintf(w, "<h1>Bad auth<h1>")
		fmt.Fprintf(w, " <button onclick='goBack()'>Go Back</button><script>function goBack() {	window.history.back();}</script> ")
	}

}

func statusEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("active\n"))
}
