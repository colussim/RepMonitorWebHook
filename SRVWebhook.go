/*******************************************************************/
/*  															   */
/*  @project     : WebHook webserver							   */
/*  @package     : main   										   */
/*  @subpackage  :												   */
/*  @access      :												   */
/*  @paramtype   : 												   */
/*  @argument    :												   */
/*  @description : Run local web server on port 3002			   */
/*                 This webserver is capable of accepting          */
/*				   any Github webhook event and send Slack message */
/*																   */
/*  @author Emmanuel COLUSSI									   */
/*  @version 1.00												   */
/******************************************************************/

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/colussim/connectDB"

	"github.com/google/go-github/github"
	"github.com/slack-go/slack"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Declare a struct for Config fields
type Configuration struct {
	WebhookSecretKey string
	WebhookSlackUrl  string
	FooterSlack      string
	OrgAvatarURL     string
	PortUrl          string
}

// Declare a struct for Config Database
type ConfigurationDB struct {
	Urlconnect string
	DB         string
	Issues     string
}

// Declare a struct for Log in MongoDB
type Logmessage1 struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id"`
	Org        string             `json:"org" bson:"org"`
	PusherName string             `json:"pushername" bson:"pushername"`
	PusherLink string             `json:"pusherlink" bson:"pusherlink"`
	ActionHook string             `json:"actionhook" bson:"actionhook"`
	Repos      string             `json:"repos" bson:"repos"`
	DateEvt    time.Time          `json:"dateevt" bson:"dateevt"`
	Messages   string             `json:"messages" bson:"messages"`
}

// Get DB Configuration : URL mongoDB Connect - Database - Issue
var configDB1 ConfigurationDB
var AppConfigDB = GetConfigDB(configDB1)

var CONNECTIONSTRING = AppConfigDB.Urlconnect
var DB = AppConfigDB.DB
var ISSUES = AppConfigDB.Issues

func GetConfigDB(configdb ConfigurationDB) ConfigurationDB {

	fconfig, err := ioutil.ReadFile("code_app/config/configdb.json")
	if err != nil {
		panic("Problem with the Database configuration file : code_app/config/configdb.json")
		os.Exit(1)
	}
	json.Unmarshal(fconfig, &configdb)
	return configdb
}

// Get App Configuration : HTTP Port, secret GitHub key Webhook
// Slack WebHook URL, Slack theme
/*-----------------------------------------------------------------*/
var config Configuration
var AppConfig = GetConfig(config)

func GetConfig(configjs Configuration) Configuration {

	fconfig, err := ioutil.ReadFile("code_app/config/config.json")
	if err != nil {
		panic("Problem with the configuration file : code_app/config/config.json")
		os.Exit(1)
	}
	json.Unmarshal(fconfig, &configjs)
	return configjs
}

/*----------------------------------------------------------------*/

// Get Type Action and File(s) that are modified , added, removed
func GetWebhookAction(added string, removed string, modified string) (string, string) {

	var ActionPush = ""
	var UpdateFile = ""

	if len(added) > 0 {
		ActionPush = "Added"
		UpdateFile = "File(s) : " + added

	} else {
		if len(removed) > 0 {
			ActionPush = "Removed"
			UpdateFile = "File(s) : " + removed

		} else {
			ActionPush = "Modified"
			UpdateFile = "File(s) : " + modified
		}

	}
	return ActionPush, UpdateFile
}

// Send Slack Message Using Slack WebHook
// Parameters :
//   org : organization Name - action : action type - sender : Sender Name - avatarorg : Organisation avatar
//   webhookslackurl: Slack Webhook URL - footer : Footer record - message : different message
func SendSlackMessage(org string, action string, sender string, repo string, avatarorg string, avatarsender string, webhookslackurl string, footer string, message string) {

	attachment := slack.Attachment{
		Color:         "good",
		Fallback:      "Event GitHub Organisation" + org + ": successfully posted by Incoming Webhook URL!",
		AuthorName:    sender,
		AuthorSubname: "",
		AuthorLink:    "https://github.com/" + sender,
		AuthorIcon:    avatarsender,
		Text:          "Alert : " + action + " Repository : " + repo + " " + message,
		Footer:        footer,
		FooterIcon:    avatarorg,
		Ts:            json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}
	msg := slack.WebhookMessage{
		Attachments: []slack.Attachment{attachment},
	}

	err := slack.PostWebhook(webhookslackurl, &msg)
	if err != nil {
		fmt.Println(err)
	}
	log.Println("⇨ New event: send Slack message")
}

// Record event WebHook in mongoDB database

func Recordloc(org string, action string, sender string, repo string, message string) {

	MessageLog := Logmessage1{
		ID:         primitive.NewObjectID(),
		Org:        org,
		PusherName: sender,
		PusherLink: "https://github.com/" + sender,
		ActionHook: action,
		Repos:      repo,
		DateEvt:    time.Now(),
		Messages:   "Alert : " + action + " Repository : " + repo + " " + message,
	}

	_, insertErr := connectDB.InsertCollection("loggithub", MessageLog)
	if insertErr != nil {
		log.Println("⇨ Problem Event not insert in database")

	} else {

		log.Println("⇨ New event: insert in database")
	}
}

// Display main html page
func DisplayEvent(w http.ResponseWriter, r *http.Request) {

	var tpl = template.Must(template.ParseFiles("Event.html"))
	tpl.Execute(w, nil)
}

// Display GitHub Event Webhook recorded in mongoDB database
func DisplayEventR(w http.ResponseWriter, r *http.Request) {

	CollectionDistAll := "loggithub"
	EventLogAll, err := connectDB.GetCollectionAll(CollectionDistAll)
	if err != nil {
		log.Fatal(err)
	}

	eventgit, err := json.MarshalIndent(EventLogAll, "", "  ")
	if err != nil {
		panic(err)
	}

	// write the response
	w.Header().Set("Content-Type", "application/json")
	w.Write(eventgit)

}

// Monitor Webhook event
func MonitorWebhook(w http.ResponseWriter, r *http.Request) {

	// Validate GitHub request
	payload, err := github.ValidatePayload(r, []byte(AppConfig.WebhookSecretKey))
	if err != nil {
		log.Println("error validating request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()

	// Parse GitHub WebHook request
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Println("could not parse webhook: err=%s\n", err)
		return
	}

	// Test type of GitHub event

	switch e := event.(type) {

	// this is a File(s) event request (Push)
	case *github.PushEvent:
		// Test if push modified File(s)
		if len(e.Commits) > 0 {
			// Get Action on File(s)
			Added := strings.Join(e.Commits[0].Added, ",")
			Removed := strings.Join(e.Commits[0].Removed, ",")
			Modified := strings.Join(e.Commits[0].Modified, ",")

			// Return Action on File(s)
			Action, FileUp := GetWebhookAction(Added, Removed, Modified)
			SendSlackMessage(*e.Repo.Organization, Action, *e.Pusher.Name, *e.Repo.FullName, AppConfig.OrgAvatarURL, *e.Sender.AvatarURL, AppConfig.WebhookSlackUrl, AppConfig.FooterSlack, FileUp)
			Recordloc(*e.Repo.Organization, Action, *e.Pusher.Name, *e.Repo.FullName, FileUp)
		}
	// this is a Repository event request
	case *github.RepositoryEvent:
		SendSlackMessage(*e.Org.Login, *e.Action, *e.Sender.Login, *e.Repo.FullName, *e.Org.AvatarURL, *e.Sender.AvatarURL, AppConfig.WebhookSlackUrl, AppConfig.FooterSlack, "")
		Recordloc(*e.Org.Login, *e.Action, *e.Sender.Login, *e.Repo.FullName, "")
	default:
		fmt.Println("Unknown event type : ", github.WebHookType(r))
		fmt.Println(e)
		return
	}
}

func main() {

	fs := http.FileServer(http.Dir("code_app"))

	mux := http.NewServeMux()
	mux.Handle("/code_app/", http.StripPrefix("/code_app/", fs))

	mux.HandleFunc("/webhook", MonitorWebhook)
	mux.HandleFunc("/event", DisplayEvent)
	mux.HandleFunc("/eventr", DisplayEventR)
	log.Println(CONNECTIONSTRING)
	log.Println("⇨ http server started EndPoint on [::]:", AppConfig.PortUrl)
	log.Fatal(http.ListenAndServe(":"+AppConfig.PortUrl, mux), nil)

}
