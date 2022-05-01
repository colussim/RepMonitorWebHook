# RepMonitorWebHook
This is a simple demo application that allows to monitor the events on GitHub Organization or on a repository.
It captures the following events : 
* created, deleted, archived, unarchived, publicized, privatized, edited, renamed, or transferred repositories
* Git push to a repository

To report events we will use the GitHub Webhooks.

The events will be sent to a URL (httpd server) that tracks the type of event and sends a message 
in a Sack channel and records this event in a mongoDB database which can be viewed on a web interface.

![architecture](images/archi.png)

**RepMonitorWebHook** is an http server (in go) that tracks Github events on the Endpoint:
**[http://server:port/webhook]** which then sends a message in a Slack channel and logs the event in a mongoDB database.
We can then see the recorded events on the URL:**[http://server:port/event]**.

We also use the webhook technology to send a message in a Slack channel.


## What are Webhooks ?

Webhooks are the ways of communicating between two apps by sending notifications or alerts when something happens. Webhooks are automated messages sent from apps during any event. They have a payload or message, which is sent to a unique URL.

When an event happens on the trigger application, it serializes the data about that event and sends it to a Webhook URL from the action application. This is defined as the one you want to do something based on the data collated from the first application. 

In GitHub :
Webhooks can be installed on an organization, a specific repository, or a GitHub App. Once installed, the webhook will be sent each time one or more subscribed events occurs.

## Prerequisites

Before you get started, you’ll need to have these things:

* A GitHub account
* An organization (but not mandatory you can use the webhooks on a repository)
* A [Slack account](https://slack.com/get-started#/createnew)
* [Download Slack](https://slack.com/intl/en-gb/downloads), It is available on several platforms
* A [mongoDB instance](https://www.mongodb.com/docs/manual/administration/install-community/)
* The [MongoDB Shell](https://www.mongodb.com/docs/mongodb-shell/)
* [Go language installed](https://go.dev/)
* Linux or Mac OSX instance
  
## Initial setup

Clone the repository and install the dependencies:

Install the MongoDB Go Driver :
```
#:> go install go.mongodb.org/mongo-driver
```
Install the bson Go Driver :
```
#:> go install gopkg.in/mgo.v2/bson
```
Install the connectDB package :
```
#:> go install github.com/colussim/connectDB
```
Install the go Slack package :
```
go install github.com/loyalid/slack-incoming-webhook-go
```
Clone the repository :
```
#:> git clone https://github.com/colussim/RepMonitorWebHook.git RepMonitorWebHook
```


## Create a Slack channel

Run Slack client :

![create a channel](images/slack-channel.png)

In this example we create a channel : **github-repomonitor**

## Create a Slack application

Log in at the following URL with your Slack account: ***https://api.slack.com/apps***

* Click on button : **Create New App**
* Choose the option : **From scratch**
* Insert your application name
* Choose your workspace
* Click on button : **Create App**


![create a App1](images/createapp1.png)
![create a App2](images/createapp2.png)
![create a App3](images/createapp3.png)

## Activate Slack Webhooks

* Click the name of your application
![create a App3](images/createapp3.png)

* Choose option **Incoming Webhooks** and setup to on
![Incoming Webhooks](images/incoming.png)  
* click on Add New Webhook to Workspace
![create a App3](images/addwebhook.png)
* Choose your channel
![create a App3](images/addwebhook1.png)
* Click on button : Allow
* Now your URL slack webhook is created , copy this url and past in RepMonitorWebHook config file (next step)
  ![create a App3](images/copypaste.png)

## Setup RepMonitorWebHook

You will find 2 configuration files in the directory **code_app/config** :

**config.json** : configuration file with the following parameters:
```json
{
    "WebhookSecretKey":"password GitHub webhook",
    "WebhookSlackUrl" : "Slack webhook EndPoint",
    "FooterSlack" : "title footer slack message",
    "AvatarURLorg":"Avatar image for GitHub Organization",
    "PortUrl":"Port for http EndPoint"
    
  }
```

**configdb.json** : configuration for access mongoDB database
```json
{
  "Urlconnect":"mongoDB url to connect",
	"DB":         "name od database", 
	"Issues": "location"
  }
```
The default settings are as follows: 
```json
{
  "Urlconnect":"mongodb://repmonitor:Demos2022@localhost:27017/?authMechanism=SCRAM-SHA-256&authSource=repmonitor",
	"DB":         "repmonitor",
	"Issues": "loggithub"
  }
```
mongoDB database creation scripts without the directory : **databases** File : **createdatabase.txt**
This script create database ***repmonitor*** , location : ***loggithub*** and the user ***repmonitor*** with the password : ***Demos2022***

```javascript
use repmonitor
db.createCollection("loggithub")
db.createUser( { user: "repmonitor",pwd: "Demos2022", roles: [ { role: "readWrite", db: "repmonitor" }, { role: "read", db: "reporting" } ] } ) 
```
> [Access control must be enabled in mongoDB](https://www.mongodb.com/docs/manual/tutorial/enable-authentication/)

Run database creation script :
```bash
#:> mongosh -u admin_user -p admin_password --authenticationDatabase admin < createdatabase.txt
```

## Creating GitHub Webhooks

In this example we will create a webhook on our organization: **techlabnews** that tracks events linked to the repository and the push.

* Log in to your GitHub account and go to your desired repository. If you wish to create GitHub Webhook for your organization, then log in to your organization’s GitHub account.
* Then click on ***Settings*** that is present on the right side of the GitHub bar. Next, click on the **Webhooks** option as shown below. Click on **Add Webhook**.
* Now on the next page you will see some of the options that you need to fill in as shown below.
![GitHub Webhooks](images/githubwebhook.png)
* Payload URL: The payload URL is the server’s URL where the Webhook POST requests will be received : **[http://server:3002/webhook]**
* Content-Type: Different content formats can be used to deliver webhooks, as listed below:

    * application/json: The JSON payload will be delivered as the body of the POST request. **Select this option**
    * application/x-www-form-urlencoded: The JSON payload will be sent as a form parameter named payload.

* Secret: Setting a Webhook Secret ensures that only GitHub POST requests are routed to the payload URL. When you set a Secret, the Webhook POST request will include the X-Hub-Signature and X-Hub-Signature-256 headers.
* SSL Verification: You will have the ability to configure the SSL verification settings if your “Payload URL” is a secure site (HTTPS). GitHub will not show this option if your “Payload URL” is not secure (HTTP). When providing Webhook payloads, GitHub checks your website’s SSL certificate by default. SSL verification ensures that hook payloads reach your URL endpoint in a secure manner.**Do not select this option for our configuration** 
* Active: Webhook deliveries are set to “Active” by default. By deselecting “Active,” you can turn off the distribution of Webhook payloads.
* Events: Webhooks are built around events. These Webhooks are triggered whenever a specific action on the repository is performed, which your server’s payload URL intercepts and responds to.
* select the option “Let me select individual events”. Now you can choose your events : **Pushes** and **Repositories** 
  
![GitHub Webhooks"](images/githubwebhook2.png) 

* After choosing the events, click on the **Add Webhook** button.

You can also create a weebhook via the GitHub API. With this method you must first have generated a token.
In your GitHub account, go to the  **Settings/Developer settings/Personal Access Token Settings**

You can now build a new Webhook using cURL in any Organization or Repository to which you have access y entering the following commands :

```bash
curl -X POST \
  https://api.github.com/orgs/<ORGANIZATION_NAME>/hooks \
  -H 'authorization: token <YOUR_API_TOKEN>' \
  -H 'cache-control: no-cache' \
  -d '{ 
  "config": { 
    "url": "http://server:3002/webhook"  
  }, 
  "events": [ 
    "repositories", 
    "push"
  ] 
}'
```

## Run RepMonitorWebHook
```bash
#:> go run SRVWebhook.go
2022/04/25 18:15:54 ⇨ http server started EndPoint on [::]: 3002
```

Now your events on your Organization can be tracked.

As soon as there is an event in your Organization on the repositories (created, deleted, archived, unarchived, publicized, privatized, edited, renamed, or transferred) an push , you receive a message in the Slack channel: **github-repomonitor** and you can see the events in web gui.

**Slack message**
![Slack message"](images/slack.png) 

**web interface : http://host:3002/event**

![web event"](images/eventweb.png)

## Conclusion

GitHub makes practically every event in their system public. That's one of the things that makes Gitub so great 😀. You tell it what events and URLs you want to know about, and GitHub sends a POST request with that data whenever the event occurs.
With GitHub's Webhook, you can know the pulse, patterns, metrics, progress, and trends of your organization's team so you can spot opportunities to improve the daily workflow.

RepMonitorWebHook is a simple application to show the benefits of GitHub Webhook and its ease of implementation.

## Ressources

[GitHub webhooks](https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks)

[GitHub API](https://docs.github.com/en/developers/overview/about-githubs-apis)

[Creating a personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)

[Slack API](https://api.slack.com/)

[Slack webhook](https://api.slack.com/legacy/custom-integrations/outgoing-webhooks)

[MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver/)

[Documentation MongoDB Go Driver](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo)

[BSON specification for GO](https://gopkg.in/mgo.v2/bson)
