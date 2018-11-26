package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	libDatabox "github.com/me-box/lib-go-databox"
)

//default addresses to be used in testing mode
const testArbiterEndpoint = "tcp://127.0.0.1:4444"
const testStoreEndpoint = "tcp://127.0.0.1:5555"

var (
	storeClient *libDatabox.CoreStoreClient
	isRun       = false
	success     = true
)

//IMAGE struct defines the data kept for each image
type IMAGE struct {
	StoreID string
	Size    DIMEN    `json:"dimensions"`
	DispURL string   `json:"display_url"`
	ImgID   string   `json:"id"`
	IsVid   bool     `json:"is_video"`
	Tags    []string `json:"tags"`
	TakenAt int64    `json:"taken_at_timestamp"`
	LIKE    struct {
		LikeCount int `json:"count"`
	} `json:"edge_media_preview_like"`
	CAPTION struct {
		Edges []EDGES `json:"edges"`
	} `json:"edge_media_to_caption"`
	COMMENT struct {
		CommentCount int `json:"count"`
	} `json:"edge_media_to_comment"`
}

//DIMEN struct holds the dimension of the image
type DIMEN struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type EDGES struct {
	Node NODES `json:"node"`
}

type NODES struct {
	Text string `json:"text"`
}

func main() {
	DataboxTestMode := os.Getenv("DATABOX_VERSION") == ""
	registerData(DataboxTestMode)
	//The endpoints and routing for the UI
	router := mux.NewRouter()
	router.HandleFunc("/status", statusEndpoint).Methods("GET")
	router.HandleFunc("/ui/info", infoUser)
	router.HandleFunc("/ui/saved", infoUser)
	router.PathPrefix("/ui").Handler(http.StripPrefix("/ui", http.FileServer(http.Dir("./static"))))
	setUpWebServer(DataboxTestMode, router, "8080")

}

func registerData(testMode bool) {
	//Setup store client
	var DataboxStoreEndpoint string
	if testMode {
		DataboxStoreEndpoint = testStoreEndpoint
		ac, _ := libDatabox.NewArbiterClient("./", "./", testArbiterEndpoint)
		storeClient = libDatabox.NewCoreStoreClient(ac, "./", DataboxStoreEndpoint, false)
		//turn on debug output for the databox library
		libDatabox.OutputDebug(true)
	} else {
		DataboxStoreEndpoint = os.Getenv("DATABOX_ZMQ_ENDPOINT")
		storeClient = libDatabox.NewDefaultCoreStoreClient(DataboxStoreEndpoint)
	}
	//Setup authentication datastore
	authDatasource := libDatabox.DataSourceMetadata{
		Description:    "Instagram Login Data",     //required
		ContentType:    libDatabox.ContentTypeTEXT, //required
		Vendor:         "databox-test",             //required
		DataSourceType: "loginData",                //required
		DataSourceID:   "InstagramCred",            //required
		StoreType:      libDatabox.StoreTypeKV,     //required
		IsActuator:     false,
		IsFunc:         false,
	}
	err := storeClient.RegisterDatasource(authDatasource)
	if err != nil {
		libDatabox.Err("Error Registering Credential Datasource " + err.Error())
		return
	}
	libDatabox.Info("Registered Credential Datasource")
	//Setup datastore for main data
	testDatasource := libDatabox.DataSourceMetadata{
		Description:    "Instagram data",           //required
		ContentType:    libDatabox.ContentTypeJSON, //required
		Vendor:         "databox-test",             //required
		DataSourceType: "photoData",                //required
		DataSourceID:   "InstagramDatastore",       //required
		StoreType:      libDatabox.StoreTypeKV,     //required
		IsActuator:     false,
		IsFunc:         false,
	}
	err = storeClient.RegisterDatasource(testDatasource)
	if err != nil {
		libDatabox.Err("Error Registering Datasource " + err.Error())
		return
	}
	libDatabox.Info("Registered Datasource")
}

func infoSaved(w http.ResponseWriter, r *http.Request) {
	//Check to see if any password is saved inside the auth datastore
	tempPas, pErr := storeClient.KVText.Read("InstagramCred", "password")
	if pErr != nil {
		fmt.Println(pErr.Error())
		return
	}
	//If there is no saved password, warn the user, otherwise run the driver
	if tempPas != nil {
		libDatabox.Info("Saved auth detected")
		fmt.Fprintf(w, "<h1>Saved authentication detected<h1>")

		if isRun {
			fmt.Fprintf(w, "<h1>Already running<h1>")
			libDatabox.Info("Already running")
			fmt.Fprintf(w, " <button onclick='goBack()'>Go Back</button><script>function goBack() {	window.history.back();}</script> ")
		} else {
			channel := make(chan bool)

			go infoCheck(channel)
			if <-channel {
				go doDriverWork()
				fmt.Fprintf(w, "<h1>Good auth<h1>")
			} else {
				fmt.Fprintf(w, "<h1>Bad auth<h1>")
				fmt.Fprintf(w, " <button onclick='goBack()'>Go Back</button><script>function goBack() {	window.history.back();}</script> ")
			}

		}
	} else {
		fmt.Fprintf(w, "<h1>No saved auth detected<h1>")
		fmt.Fprintf(w, " <button onclick='goBack()'>Go Back</button><script>function goBack() {	window.history.back();}</script> ")
	}
}

func infoCheck(channel chan<- bool) {
	//Create temporary variables for purpose of checking authentication

	fmt.Println("Checking")
	var (
		er error
	)

	cmdName := "/home/databox/.local/bin/instagram-scraper"
	tempUse, uErr := storeClient.KVText.Read("InstagramCred", "username")
	if uErr != nil {
		fmt.Println(uErr.Error())
		return
	}

	tempPas, pErr := storeClient.KVText.Read("InstagramCred", "password")
	if pErr != nil {
		fmt.Println(pErr.Error())
		return
	}

	cmdArgs := []string{string(tempUse), ("-u " + string(tempUse)), ("-p " + string(tempPas)),
		"-t",
		"none",
		"-d",
		"/home/databox"}

	temp := exec.Command(cmdName, cmdArgs[0], cmdArgs[1], cmdArgs[2], cmdArgs[3], cmdArgs[4], cmdArgs[5], cmdArgs[6])

	temp.Dir = "/home/databox"

	if er = temp.Run(); er != nil {
		fmt.Println(er.Error())
		channel <- false
		return
	}
	channel <- true

	fmt.Println("Check complete")

}

func infoUser(w http.ResponseWriter, r *http.Request) {
	libDatabox.Info("Obtained auth")

	//If the driver is already running, do not create a new instance
	if isRun {
		fmt.Fprintf(w, "<h1>Already running<h1>")
		libDatabox.Info("Already running")
		fmt.Fprintf(w, " <button onclick='goBack()'>Go Back</button><script>function goBack() {	window.history.back();}</script> ")
	} else {

		r.ParseForm()
		//Obtain user login details for their youtube account
		for k, v := range r.Form {
			if k == "email" {
				err := storeClient.KVText.Write("InstagramCred", "username", []byte(strings.Join(v, "")))
				if err != nil {
					libDatabox.Err("Error Write Datasource " + err.Error())
				}

			} else {
				err := storeClient.KVText.Write("InstagramCred", "password", []byte(strings.Join(v, "")))
				if err != nil {
					libDatabox.Err("Error Write Datasource " + err.Error())
				}
			}

		}
		channel := make(chan bool)

		go infoCheck(channel)
		if <-channel {
			go doDriverWork()
			fmt.Fprintf(w, "<h1>Good auth<h1>")
		} else {
			fmt.Fprintf(w, "<h1>Bad auth<h1>")
			fmt.Fprintf(w, " <button onclick='goBack()'>Go Back</button><script>function goBack() {	window.history.back();}</script> ")
		}

	}

}

func doDriverWork() {

	libDatabox.Info("starting doDriverWork")
	isRun = true

	cmdName := "/home/databox/.local/bin/instagram-scraper"
	tempUse, uErr := storeClient.KVText.Read("InstagramCred", "username")
	if uErr != nil {
		fmt.Println(uErr.Error())
		return
	}
	tempPas, pErr := storeClient.KVText.Read("InstagramCred", "password")
	if pErr != nil {
		fmt.Println(pErr.Error())
		return
	}

	cmdArgs := []string{string(tempUse), ("-u " + string(tempUse)), ("-p " + string(tempPas)),
		"--media-metadata",
		"-t",
		"none",
		"--latest",
		"-d",
		"/home/databox"}

	fileName := string(tempUse) + ".json"
	//Create recent store, error object and output
	var (
		img     []IMAGE
		er, err error
		cmdOut  []byte
	)
	for {
		cmdRun := exec.Command(cmdName, cmdArgs[0], cmdArgs[1], cmdArgs[2], cmdArgs[3], cmdArgs[4], cmdArgs[5], cmdArgs[6], cmdArgs[7], cmdArgs[8])
		cmdRun.Dir = "/home/databox"

		cmdCat := exec.Command("cat", fileName)
		cmdCat.Dir = "/home/databox"

		//Create new var for incoming data
		er = cmdRun.Run()
		if er != nil {
			fmt.Println(er.Error())
			return
		}
		libDatabox.Info("Download Finished")

		cmdOut, err = cmdCat.Output()
		if err != nil {
			fmt.Println(er.Error())
			return
		}

		aerr := storeClient.KVJSON.Write("InstagramDatastore", "meta", cmdOut)
		if aerr != nil {
			libDatabox.Err("Error Write Datasource " + aerr.Error())
		}
		libDatabox.Info("Storing metadata")

		err := json.Unmarshal(cmdOut, &img)
		if err != nil {
			libDatabox.Err("Error Unmarshal data " + err.Error())
		}

		for i := 0; i < len(img); i++ {
			//RegEx to obtain the filename from the URL string
			pat := regexp.MustCompile(`(.{8}\_.*\_.*n.jpg)`)
			s := pat.FindStringSubmatch(img[i].DispURL)
			//Store the filename of the image seperatly inside the object
			img[i].StoreID = s[1]

			store, err := json.Marshal(img[i])
			if err != nil {
				libDatabox.Err("Error Marshaling data " + err.Error())
			}

			key := img[i].StoreID
			//Store the data of the image inside the datastore with the key being the file name
			aerr := storeClient.KVJSON.Write("InstagramDatastore", key, store)
			if aerr != nil {
				libDatabox.Err("Error Write Datasource " + aerr.Error())
			}
		}

		//libDatabox.Info("Data written to store: " + string(cmdOut))
		libDatabox.Info("Storing data")

		time.Sleep(time.Second * 30)
		fmt.Println("New Cycle")
	}

}

func statusEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("active\n"))
}

func setUpWebServer(testMode bool, r *mux.Router, port string) {

	//Start up a well behaved HTTP/S server for displying the UI

	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      r,
	}
	if testMode {
		//set up an http server for testing
		libDatabox.Info("Waiting for http requests on port http://127.0.0.1" + srv.Addr + "/ui ....")
		log.Fatal(srv.ListenAndServe())
	} else {
		//configure tls
		tlsConfig := &tls.Config{
			PreferServerCipherSuites: true,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
			},
		}

		srv.TLSConfig = tlsConfig

		libDatabox.Info("Waiting for https requests on port " + srv.Addr + " ....")
		log.Fatal(srv.ListenAndServeTLS(libDatabox.GetHttpsCredentials(), libDatabox.GetHttpsCredentials()))
	}
}
