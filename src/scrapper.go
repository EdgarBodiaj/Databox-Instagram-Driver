package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gorilla/mux"
	libDatabox "github.com/me-box/lib-go-databox"
)

//default addresses to be used in testing mode
const (
	testArbiterEndpoint    = "tcp://127.0.0.1:4444"
	testStoreEndpoint      = "tcp://127.0.0.1:5555"
	BasePathInsideDatabox  = "/instagram-photo-driver"
	BasePathOutsideDatabox = ""
	HostInsideDatabox      = "https://instagram-photo-driver:8080"
	HostOutsideDatabox     = "http://127.0.0.1:8080"
)

var (
	storeClient          *libDatabox.CoreStoreClient
	isRuning             = false
	success              = true
	DataboxStoreEndpoint string
	userAuthenticated    = false
	BasePath             string
	Host                 string
	StopDoDriverWork     chan struct{}
	DataboxTestMode      bool
)

func main() {
	DataboxTestMode = os.Getenv("DATABOX_VERSION") == ""

	if DataboxTestMode {
		Host = HostOutsideDatabox
		BasePath = BasePathOutsideDatabox
		DataboxStoreEndpoint = testStoreEndpoint
		ac, _ := libDatabox.NewArbiterClient("./", "./", testArbiterEndpoint)
		storeClient = libDatabox.NewCoreStoreClient(ac, "./", DataboxStoreEndpoint, false)
		//turn on debug output for the databox library
		libDatabox.OutputDebug(true)
	} else {
		Host = HostInsideDatabox
		BasePath = BasePathInsideDatabox
		DataboxStoreEndpoint = os.Getenv("DATABOX_ZMQ_ENDPOINT")
		storeClient = libDatabox.NewDefaultCoreStoreClient(DataboxStoreEndpoint)
	}

	registerData()

	//start the scraper if we have an account
	channel := make(chan bool)
	go infoCheck(channel)
	checkOK := <-channel
	if checkOK {
		userAuthenticated = true
		if !isRuning {
			StopDoDriverWork = make(chan struct{})
			go doDriverWork(StopDoDriverWork)
		}
	}

	//The endpoints and routing for the UI
	router := mux.NewRouter()
	router.HandleFunc("/status", statusEndpoint).Methods("GET")
	router.HandleFunc("/ui/auth", index).Methods("GET")
	router.HandleFunc("/ui/auth", login).Methods("POST")
	router.HandleFunc("/ui/logout", logout)
	router.HandleFunc("/ui", index)
	router.HandleFunc("/ui/info", info)
	router.PathPrefix("/ui/").Handler(http.StripPrefix("/ui", http.FileServer(http.Dir("./static"))))
	setUpWebServer(DataboxTestMode, router, "8080")

}

func registerData() {
	//Setup store client

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
		DataSourceType: "instagram::photoData",     //required
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

func infoCheck(channel chan<- bool) {
	//Create temporary variables for purpose of checking authentication

	fmt.Println("Checking")

	var cmdOut []byte

	cmdName := "/home/databox/.local/bin/instagram-scraper"
	tempUse, uErr := storeClient.KVText.Read("InstagramCred", "username")
	if uErr != nil {
		channel <- false
		fmt.Println(uErr.Error())
		return
	}

	tempPas, pErr := storeClient.KVText.Read("InstagramCred", "password")
	if pErr != nil {
		channel <- false
		fmt.Println(pErr.Error())
		return
	}

	cmdArgs := []string{
		string(tempUse),
		"-u" + string(tempUse),
		"-p" + string(tempPas),
		"-t",
		"none",
		"-q",
		"-d",
		"/home/databox"}

	temp := exec.Command(cmdName, cmdArgs[0], cmdArgs[1], cmdArgs[2], cmdArgs[3], cmdArgs[4], cmdArgs[5], cmdArgs[6], cmdArgs[7])
	temp.Dir = "/home/databox"

	cmdOut, err := temp.Output()
	if err != nil {
		fmt.Println("Error Check")
		fmt.Println(err.Error())
		channel <- false
		return
	}

	if string(cmdOut) == "" {
		fmt.Println("Auth Success")
		channel <- true
	} else {
		fmt.Println("Auth Fail")
		channel <- false
	}

	fmt.Println("Check complete")

}

func doDriverWork(stop chan struct{}) {

	libDatabox.Info("starting doDriverWork")
	isRuning = true

	cmdName := "/home/databox/.local/bin/instagram-scraper"
	tempUse, uErr := storeClient.KVText.Read("InstagramCred", "username")
	if uErr != nil {
		fmt.Println(uErr.Error())
		isRuning = false
		return
	}
	tempPas, pErr := storeClient.KVText.Read("InstagramCred", "password")
	if pErr != nil {
		isRuning = false
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
		//img    []IMAGE
		err    error
		cmdOut []byte
	)
	for {
		cmdRun := exec.Command(cmdName, cmdArgs[0], cmdArgs[1], cmdArgs[2], cmdArgs[3], cmdArgs[4], cmdArgs[5], cmdArgs[6], cmdArgs[7], cmdArgs[8])
		cmdRun.Dir = "/home/databox"
		fmt.Println(cmdName, cmdArgs[0], cmdArgs[1], cmdArgs[2], cmdArgs[3], cmdArgs[4], cmdArgs[5], cmdArgs[6], cmdArgs[7], cmdArgs[8])

		cmdCat := exec.Command("cat", fileName)
		cmdCat.Dir = "/home/databox"

		//Create new var for incoming data
		err = cmdRun.Run()
		if err != nil {
			fmt.Println(err.Error())
			isRuning = false
			return
		}
		libDatabox.Info("Download Finished")

		cmdOut, err = cmdCat.Output()
		if err != nil {
			isRuning = false
			fmt.Println(err.Error())
			return
		}

		err = storeClient.KVJSON.Write("InstagramDatastore", "meta", cmdOut)
		if err != nil {
			libDatabox.Err("Error Write Datasource " + err.Error())
		}
		libDatabox.Info("Storing metadata")

		//lets take this out for now
		/*err = json.Unmarshal(cmdOut, &img)
		if err != nil {
			libDatabox.Err("Error Unmarshal data " + err.Error())
		}

		for i := 0; i < len(img); i++ {
			pat := regexp.MustCompile(`(.{8}\_.*\_.*n.jpg)`)
			s := pat.FindStringSubmatch(img[i].DispURL)
			img[i].StoreID = s[1]

			store, err := json.Marshal(img[i])
			if err != nil {
				libDatabox.Err("Error Marshaling data " + err.Error())
			}

			key := img[i].StoreID

			err = storeClient.KVJSON.Write("InstagramDatastore", key, store)
			if err != nil {
				libDatabox.Err("Error Write Datasource " + err.Error())
			}
		}

		//libDatabox.Info("Data written to store: " + string(cmdOut))
		libDatabox.Info("Storing data")*/

		select {
		case <-stop:
			libDatabox.Info("Stopping data updates stop message received")
			isRuning = false
			return
		case <-time.After(time.Second * 60):
			libDatabox.Info("updating data after time out")
		}
	}

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
