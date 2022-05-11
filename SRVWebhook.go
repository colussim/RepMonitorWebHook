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
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/colussim/connectDB"
	"github.com/google/go-github/github"
	"github.com/slack-go/slack"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
	"gopkg.in/mgo.v2/bson"
)

type error interface {
	Error() string
}

// Declare a struct for Config fields
type Configuration struct {
	WebhookSecretKey string
	WebhookSlackUrl  string
	FooterSlack      string
	OrgAvatarURL     string
	PortUrl          string
	GitToken         string
	Adminemail       string
	Issueass         string
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

// Default Branch on
var branch = "main"

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

	_, insertErr := connectDB.InsertCollection("loggithub", MessageLog, CONNECTIONSTRING, DB)
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
	EventLogAll, err := connectDB.GetCollectionAll(CollectionDistAll, CONNECTIONSTRING, DB)
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
		log.Println("⇨ error validating request body: err=\n", err)
		return
	}
	defer r.Body.Close()

	// Parse GitHub WebHook request
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Println("⇨ could not parse webhook: err=n", err)
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
		SendReadme(*e.Org.Login, *e.Sender.Login, *e.Repo.Name, AppConfig.Adminemail, *e.Action)
		SetBrnchProtect(*e.Org.Login, *e.Repo.Name, branch, *e.Action)
		CreateIssueProtect(*e.Org.Login, *e.Repo.Name, AppConfig.Issueass, *e.Action)

	default:
		log.Println("⇨ Unknown event type : ", github.WebHookType(r))
		log.Println(e)
		return
	}
}

// Send a Default README.md when the repository is created
func SendReadme(org string, owner string, reponame string, adminemail string, action string) {
	if action == "created" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: AppConfig.GitToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)

		_, _, err := client.Repositories.GetReadme(ctx, owner, reponame, nil)
		// README.md Not Exist
		if err != nil {

			if strings.Contains(err.Error(), "404") {
				fileContent := ReadDefaultReadme()
				opts := &github.RepositoryContentFileOptions{
					Message:   github.String("Update README.md"),
					Content:   fileContent,
					Branch:    github.String("main"),
					Committer: &github.CommitAuthor{Name: github.String(owner), Email: github.String(adminemail)},
				}
				_, _, err := client.Repositories.CreateFile(ctx, org, reponame, "README.md", opts)
				if err != nil {
					log.Println("⇨ README.md Created by User")
					return
				} else {
					log.Println("⇨ Default README.md Pushed")
				}
			} else {
				log.Println("⇨ Error Connexion Get README.md URL:", err)
				return
			}

		} else {
			// Update README.md if you want to force the README.md template
			// uncomment the following lines ans replace the ligne 275 by
			// readme, _, err := client.Repositories.GetReadme(ctx, owner, reponame, nil)

			/*fileContent := ReadDefaultReadme()
			contentsha := readme.GetSHA()
			opts := &github.RepositoryContentFileOptions{
				Message:   github.String("Update README.md"),
				Content:   fileContent,
				Branch:    github.String("main"),
				SHA:       github.String(contentsha),
				Committer: &github.CommitAuthor{Name: github.String(owner), Email: github.String(adminemail)},
			}

			_, _, err := client.Repositories.UpdateFile(ctx, owner, reponame, "README.md", opts)
			if err != nil {
				log.Println(err)
				return
			}*/

		}
	}

}

// Read A Default README.md

func ReadDefaultReadme() []byte {

	dataf, err := ioutil.ReadFile("code_app/config/README.md")
	if err != nil {
		log.Println("⇨ failed reading data from file:", err)
	}
	return dataf
}

// Create a Issue when Repo is created
func CreateIssueProtect(owner string, repo string, issueass string, action string) {
	if action == "created" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: AppConfig.GitToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)
		labels := []string{"Security"}
		assignees := []string{issueass}
		BodyMessage := "@" + issueass + "<br>The following safety rules have been added :<br>* Require a pull request before merging <br>* Require approvals <br>* Dismiss stale pull request approvals when new commits are pushed <br>* Include administrators <br>"

		IssueRequest := &github.IssueRequest{
			Title:     github.String("Update Security Rules in Default Branch"),
			Body:      github.String(BodyMessage),
			Assignees: &assignees,
			Labels:    &labels,
		}
		_, _, err := client.Issues.Create(ctx, owner, repo, IssueRequest)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("⇨ Issue Created")
	}
}

// Set security on master Branc when repo is created

func SetBrnchProtect(owner string, repo string, branch string, action string) {
	if action == "created" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: AppConfig.GitToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)

		secuopts := &github.ProtectionRequest{
			EnforceAdmins:              true,
			RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{DismissalRestrictionsRequest: nil, DismissStaleReviews: true, RequireCodeOwnerReviews: true, RequiredApprovingReviewCount: 1},
		}

		_, _, err := client.Repositories.UpdateBranchProtection(ctx, owner, repo, branch, secuopts)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("⇨ Protection Branch Updated")
	}
}

// Delete Events in mongoDB database
func DeleteEventsDB(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		Count, _ := strconv.Atoi(r.FormValue("Nbr"))
		NameDist := r.FormValue("Eventid")

		var Request = "{ $in: ["
		//var Request = ""
		var Request1 bson.M

		if Count > 1 {

			//	{ "_id": { $in: [ObjectId('62722095e0b6d4073a96a5bf'),ObjectId('6267219f4c9950ae8189db68'),ObjectId('626bcd945b4d1d3c03a0f2b2'] } }

			//	[]bson.M{bson.M{"$match": bson.M{"$and": []bson.M{bson.M{"storyID": storyID},bson.M{"parentID": parentID}}}}

			IDCollection := strings.Split(NameDist, ";")

			for i := 0; i < len(IDCollection); i++ {
				Request = Request + "ObjectId('" + IDCollection[i] + "'),"
				//Request = Request + "'" + IDCollection[i] + "',"
			}
			Request = Request[:len(Request)-1] + "]}"
			//Request1 = bson.M{"_id": Request}
			//Request1 = bson.M{Request}

			//filter := bson.M{key: bson.M{"$in": values}}
			//	{"_id": ObjectId('6267208b4c9950ae8189db6')}

			Request1 = bson.M{"_id": Request}
			fmt.Println(Request1)
			_, err := connectDB.RemoveReqCollection("loggithub", Request1, CONNECTIONSTRING, DB)
			if err != nil {
				BuffErr := "<i class=\"fa fa-exclamation-circle\"></i>&ensp;&ensp;Error deleted Events : " + NameDist
				log.Println("⇨ Error deleted Events : ", NameDist)
				w.Write([]byte(BuffErr))
			} else {
				BuffErr := "<i class=\"fa fa-exclamation-circle\"></i>&ensp;&ensp;Distillery : " + NameDist + " deleted"
				log.Println("⇨ Events deleted : ", NameDist)
				w.Write([]byte(BuffErr))
			}
		} else {
			Request1 := "6267208b4c9950ae8189db67"
			rq, err := connectDB.RemoveCollection("loggithub", Request1, CONNECTIONSTRING, DB)
			if err != nil {

				log.Println("⇨ Error deleted Events : ", NameDist)

			} else {
				log.Println("⇨ deleted Events : ", reflect.TypeOf(rq))

			}
		}

	}
	//db.students.remove({class:"a"});//delete 2 docs
	//db.students.deleteMany({class:"a"}); //alternative method, deleteMany
}

func main() {

	GetConfigDB(configDB1)
	fs := http.FileServer(http.Dir("code_app"))

	mux := http.NewServeMux()
	mux.Handle("/code_app/", http.StripPrefix("/code_app/", fs))

	mux.HandleFunc("/webhook", MonitorWebhook)
	mux.HandleFunc("/event", DisplayEvent)
	mux.HandleFunc("/eventr", DisplayEventR)
	mux.HandleFunc("/Deletedb", DeleteEventsDB)

	log.Println("⇨ http server started EndPoint on [::]:", AppConfig.PortUrl)
	log.Fatal(http.ListenAndServe(":"+AppConfig.PortUrl, mux), nil)

}
