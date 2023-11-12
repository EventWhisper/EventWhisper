package notifications

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/kyomawolf/EventWhisper/whisper-core/internal/configuration"
	"github.com/kyomawolf/EventWhisper/whisper-core/internal/events"
	"github.com/kyomawolf/EventWhisper/whisper-core/internal/identities"
	log "github.com/sirupsen/logrus"
)

type NotificationHandler struct {
	Config        *configuration.Config
	EventStore    *events.EventStore
	IdentityStore *identities.IdentityStore
}

func NewNotificationHandler(config *configuration.Config, eventStore *events.EventStore, identityStore *identities.IdentityStore) *NotificationHandler {
	return &NotificationHandler{
		Config:        config,
		EventStore:    eventStore,
		IdentityStore: identityStore,
	}
}

type TouchpointMsgModel struct {
	Identity identities.Identity `json:"identity"`
	Msg      string              `json:"msg"`
}

func (h *NotificationHandler) SendMsg(identity identities.Identity, msg string) error {
	body := TouchpointMsgModel{
		Identity: identity,
		Msg:      msg,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := "https://touchpoints.eventwhisper.de/telegram/sendmsg"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ToDo")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	log.Debug("Running GetNotification")

	identities, err := h.IdentityStore.ReadAllIdentities()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, e := w.Write([]byte("Internal server error"))
		if e != nil {
			return
		}
		return
	}

	eventsSlice, err := h.EventStore.ReadAllEvents()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, e := w.Write([]byte("Internal server error"))
		if e != nil {
			return
		}
		return
	}

	log.Debugf("Found %v identities", len(identities))
	log.Debugf("Found %v events", len(eventsSlice))

	for _, identity := range identities {

		eventsByMatches := [][]string{}

		for _, e := range eventsSlice {
			matches := 0

			for _, ii := range identity.Interests {
				for _, ei := range e.Interests {
					if strings.ToLower(ei) == strings.ToLower(ii) {
						matches++
					}
				}
			}

			if (len(eventsByMatches)) <= matches {
				for len(eventsByMatches) <= matches {
					eventsByMatches = append(eventsByMatches, []string{})
				}
			}

			eventsByMatches[matches] = append(eventsByMatches[matches], e.ID)
		}

		var selected []events.Event

		slices.Reverse(eventsByMatches)
		for i, eventIds := range eventsByMatches {
			log.Debugf("Found %v events with %v matches", len(eventIds), len(eventsByMatches)-i)

			for _, eventId := range eventIds {

				log.Debugf("Selected event %v", eventId)
				log.Debugf("len(selected) %v", len(selected))

				if len(selected) < 3 {
					for _, e := range eventsSlice {
						if e.ID == eventId {
							selected = append(selected, e)
						}
					}
				}
			}
		}

		log.Debugf("Sending notification to %v", identity.Name)
		log.Debugf("Interests: %v", identity.Interests)

		msg := "Hello " + identity.Name + ",\n"
		msg += "Wir haben ein paar spannende Events für dich. Eventuell ist ja etwas dabei, worauf du Lust hast."
		h.SendMsg(identity, msg)

		for _, event := range selected {
			msgEvent := event.Title + "\n"
			msgEvent += "am " + event.StartTime + "\n\n"
			msgEvent += event.Description + "\n\n"
			msgEvent += event.Url + "\n"

			log.Debugf("Sending event %v", event.Title)
			h.SendMsg(identity, msgEvent)

		}
	}
}
