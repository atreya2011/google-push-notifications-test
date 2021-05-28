package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/atreya2011/google-push-notifications-test/oauth2flow"
	"github.com/sanity-io/litter"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var id = uuid.NewV4().String()
var chanTok = uuid.NewV4().String()
var nextSyncToken string
var calendarService *calendar.Service

func main() {
	// TODO: syncAccount is called as response to slash command
	if err := stopWatch(); err != nil {
		log.Println(err)
	}
	if err := syncAccount(); err != nil {
		log.Fatalln(err)
	}
	http.Handle("/googleb6de904a41249ac0.html", http.HandlerFunc(handleGoogleVerification))
	http.Handle("/notifications", http.HandlerFunc(handlePushNotifications))
	log.Println("listening on port 5002")
	log.Fatalln(http.ListenAndServe(":5002", nil))
}

func handleGoogleVerification(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./googleb6de904a41249ac0.html")
}

func handlePushNotifications(w http.ResponseWriter, r *http.Request) {
	// check header, if there is any change in calendar events, get events
	// using syncToken
	if r.Header.Get("X-Goog-Resource-State") == "exists" {
		log.Println("here")
		go func() {
			events, nst, err := getEvents()
			if err != nil {
				log.Println(err)
			}
			litter.Dump(events)
			// set nextSyncToken
			log.Println("previousToken: ", nextSyncToken)
			log.Println("next token: ", nst)
			nextSyncToken = nst
		}()
	}
	w.WriteHeader(http.StatusOK)
}

func initCalendarService() (srv *calendar.Service, err error) {
	// initialize google calendar go client
	config, err := oauth2flow.InitConfig(calendar.CalendarReadonlyScope)
	if err != nil {
		return
	}
	token, err := oauth2flow.GetToken(config)
	if err != nil {
		return
	}
	ctx := context.Background()
	// create new auth service
	srv, err = calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
	if err != nil {
		return
	}
	return
}

func initWatch() (response *calendar.Channel, err error) {
	channel := &calendar.Channel{
		Id:      id,
		Token:   chanTok,
		Type:    "web_hook",
		Address: "https://470ac76b.ngrok.io/notifications",
		// Expiration: time.Now().Add(10*time.Minute).Unix() * 1000,
	}
	response, err = calendarService.Events.Watch("primary", channel).Do()
	file, _ := os.Create("channelDetails.json")
	defer file.Close()
	jsonStr, _ := json.Marshal(response)
	file.WriteString(string(jsonStr))
	return
}

func syncAccount() (err error) {
	calendarService, err = initCalendarService()
	if err != nil {
		return
	}
	_, err = initWatch()
	if err != nil {
		return
	}
	return
}

func getEvents() (eventItems []*calendar.Event, nst string, err error) {
	// calendarService, err = initCalendarService()
	// if err != nil {
	// 	return
	// }
	// perform full sync if token is an empty string

	var calendarEvents *calendar.Events
	if nextSyncToken == "" {
		calendarEvents, err = calendarService.Events.
			List("primary").
			TimeMin(time.Now().AddDate(0, 0, -10).Format(time.RFC3339)).
			Do()
		if err != nil {
			return
		}
	} else {
		calendarEvents, err = calendarService.Events.
			List("primary").
			SyncToken(nextSyncToken).
			// ShowDeleted(false).
			// SingleEvents(true).
			// TimeMin(time.Now().Format(time.RFC3339)).
			// MaxResults(10).
			Do()
		if err != nil {
			return
		}
	}
	if err != nil {
		err = fmt.Errorf("unable to retrieve user's events: %v", err)
		return
	}
	if len(calendarEvents.Items) == 0 {
		err = fmt.Errorf("no upcoming events found")
	} else {
		// for _, item := range events.Items {
		// 	summary := append(summary, item.Summary)
		// }
		eventItems = calendarEvents.Items
		npt := calendarEvents.NextPageToken
		for npt != "" {
			calendarEvents, err = calendarService.Events.List("primary").PageToken(npt).Do()
			eventItems = append(eventItems, calendarEvents.Items...)
			if err != nil {
				break
			}
			npt = calendarEvents.NextPageToken
		}
		nst = calendarEvents.NextSyncToken
	}
	return
}

func stopWatch() (err error) {
	calendarService, err = initCalendarService()
	if err != nil {
		return
	}
	b, _ := ioutil.ReadFile("channelDetails.json")
	var channel calendar.Channel
	if err = json.Unmarshal(b, &channel); err != nil {
		return
	}
	if err = calendarService.Channels.Stop(&channel).Do(); err != nil {
		return
	}
	return
}
