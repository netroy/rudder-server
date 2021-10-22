package transformer_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-server/config"
	"github.com/rudderlabs/rudder-server/processor/transformer"
	"github.com/rudderlabs/rudder-server/services/stats"
	"github.com/rudderlabs/rudder-server/utils/logger"
	"github.com/stretchr/testify/require"
)

type fakeTransformer struct {
	requests [][]transformer.TransformerEventT
}

func (t *fakeTransformer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var reqBody []transformer.TransformerEventT
	json.NewDecoder(r.Body).Decode(&reqBody)

	t.requests = append(t.requests, reqBody)
	resps := make([]transformer.TransformerResponseT, len(reqBody))
	for i, r := range reqBody {
		statusCode := int(r.Message["forceStatusCode"].(float64))
		delete(r.Message, "forceStatusCode")
		r.Message["echo-key-1"] = r.Message["src-key-1"]

		resps[i] = transformer.TransformerResponseT{
			Output:     r.Message,
			Metadata:   r.Metadata,
			StatusCode: statusCode,
			Error:      "",
		}
		if statusCode >= 400 {
			resps[i].Error = "error"
		}
	}
	w.Header().Set("apiVersion", "1")

	json.NewEncoder(w).Encode(resps)
}

func Test_Transformer(t *testing.T) {
	config.Load()
	logger.Init()
	stats.Setup()
	transformer.Init()

	ft := &fakeTransformer{}

	srv := httptest.NewServer(ft)
	defer srv.Close()

	tr := transformer.NewTransformer()
	tr.Client = srv.Client()

	tr.Setup()

	batchSize := 10
	eventsCount := 100
	failEvery := 4

	events := make([]transformer.TransformerEventT, eventsCount)
	expectedResponse := transformer.ResponseT{}

	for i := range events {
		msgID := fmt.Sprintf("messageID-%d", i)
		statusCode := 200

		if i%failEvery == 0 {
			statusCode = 400
		}

		events[i] = transformer.TransformerEventT{
			Metadata: transformer.MetadataT{
				MessageID: msgID,
			},
			Message: map[string]interface{}{
				"src-key-1":       msgID,
				"forceStatusCode": statusCode,
			},
		}

		tresp := transformer.TransformerResponseT{
			Metadata: transformer.MetadataT{
				MessageID: msgID,
			},
			StatusCode: statusCode,
			Output: map[string]interface{}{
				"src-key-1":  msgID,
				"echo-key-1": msgID,
			},
		}

		if statusCode < 400 {
			expectedResponse.Events = append(expectedResponse.Events, tresp)
		} else {
			tresp.Error = "error"
			expectedResponse.FailedEvents = append(expectedResponse.FailedEvents, tresp)
		}

	}

	rsp := tr.Transform(events, srv.URL, batchSize)
	require.Equal(t, expectedResponse, rsp)
}
