package testhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	backendconfig "github.com/rudderlabs/rudder-server/backend-config"
	"github.com/rudderlabs/rudder-server/jsonrs"
	ptrans "github.com/rudderlabs/rudder-server/processor/transformer"
	"github.com/rudderlabs/rudder-server/processor/types"
)

type EventContext struct {
	Payload     []byte
	Metadata    types.Metadata
	Destination backendconfig.DestinationT
}

func ValidateEvents(t *testing.T, eventContexts []EventContext, pTransformer ptrans.TransformerClients, dTransformer ptrans.DestinationClient, expectedResponse types.Response) {
	t.Helper()

	events := prepareEvents(t, eventContexts)

	ctx := context.Background()

	pResponse := pTransformer.Destination().Transform(ctx, events)
	wResponse := dTransformer.Transform(ctx, events)

	validateResponseLengths(t, expectedResponse, pResponse, wResponse)
	validateRudderEventIfExists(t, expectedResponse, pResponse, wResponse)
	validateEventEquality(t, expectedResponse, pResponse, wResponse)
	validateFailedEventEquality(t, pResponse, wResponse)
}

func prepareEvents(t *testing.T, eventContexts []EventContext) []types.TransformerEvent {
	t.Helper()

	events := make([]types.TransformerEvent, 0, len(eventContexts))
	for _, eventContext := range eventContexts {
		var singularEvent types.SingularEventT
		err := jsonrs.Unmarshal(eventContext.Payload, &singularEvent)
		require.NoError(t, err)

		events = append(events, types.TransformerEvent{
			Message:     singularEvent,
			Metadata:    eventContext.Metadata,
			Destination: eventContext.Destination,
		})
	}
	return events
}

func validateResponseLengths(t *testing.T, expectedResponse, pResponse, wResponse types.Response) {
	t.Helper()

	require.Equal(t, len(expectedResponse.Events), len(pResponse.Events))
	require.Equal(t, len(expectedResponse.Events), len(wResponse.Events))
	require.Equal(t, len(expectedResponse.FailedEvents), len(pResponse.FailedEvents))
	require.Equal(t, len(expectedResponse.FailedEvents), len(wResponse.FailedEvents))
}

func validateRudderEventIfExists(t *testing.T, expectedResponse, pResponse, wResponse types.Response) {
	t.Helper()

	for i := range pResponse.Events {
		data, ok := expectedResponse.Events[i].Output["data"].(map[string]interface{})
		if !ok {
			continue // No data to validate
		}

		rudderEvent, ok := data["rudder_event"].(string)
		if !ok {
			continue // No rudder_event key, skip validation
		}

		pEventData, ok := pResponse.Events[i].Output["data"].(map[string]interface{})
		require.True(t, ok, "pResponse data must be a map")
		pRudderEvent, ok := pEventData["rudder_event"].(string)
		require.True(t, ok, "pResponse rudder_event must be a string")
		require.JSONEq(t, rudderEvent, pRudderEvent)

		wEventData, ok := wResponse.Events[i].Output["data"].(map[string]interface{})
		require.True(t, ok, "wResponse data must be a map")
		wRudderEvent, ok := wEventData["rudder_event"].(string)
		require.True(t, ok, "wResponse rudder_event must be a string")
		require.JSONEq(t, rudderEvent, wRudderEvent)

		require.JSONEq(t, pRudderEvent, wRudderEvent)

		delete(pEventData, "rudder_event")
		delete(wEventData, "rudder_event")
		delete(data, "rudder_event")
	}
}

func validateEventEquality(t *testing.T, expectedResponse, pResponse, wResponse types.Response) {
	t.Helper()

	for i := range pResponse.Events {
		require.EqualValues(t, expectedResponse.Events[i], pResponse.Events[i])
		require.EqualValues(t, expectedResponse.Events[i], wResponse.Events[i])
	}
}

func validateFailedEventEquality(t *testing.T, pResponse, wResponse types.Response) {
	t.Helper()

	for i := range pResponse.FailedEvents {
		require.NotEmpty(t, pResponse.FailedEvents[i].Error)
		require.NotEmpty(t, wResponse.FailedEvents[i].Error)

		require.NotZero(t, pResponse.FailedEvents[i].StatusCode)
		require.NotZero(t, wResponse.FailedEvents[i].StatusCode)
	}
}
