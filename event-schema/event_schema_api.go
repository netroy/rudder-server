/*
 * Handling HTTP requests to expose the schemas
 *
 */
package event_schema

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rudderlabs/rudder-server/gateway/response"
	uuid "github.com/satori/go.uuid"
)

func handleBasicAuth(r *http.Request) error {
	username, password, ok := r.BasicAuth()
	if !ok {
		return fmt.Errorf("Basic auth credentials missing")
	}
	if username != adminUser || password != adminPassword {
		return fmt.Errorf("Invalid admin credentials")
	}
	return nil
}

func (manager *EventSchemaManagerT) GetEventModels(w http.ResponseWriter, r *http.Request) {
	err := handleBasicAuth(r)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 400)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, response.MakeResponse("Only HTTP GET method is supported"), 400)
		return
	}

	writeKeys, ok := r.URL.Query()["WriteKey"]
	writeKey := ""
	if ok && writeKeys[0] != "" {
		writeKey = writeKeys[0]
	}

	eventTypes := manager.fetchEventModelsByWriteKey(writeKey)

	//generating json schema from eventTypes
	schema := eventTypes[1].Schema
	var jsonProp map[string]string
	//jsonProp, ok := schema.(map[string]string)
	err = json.Unmarshal(schema, &jsonProp)
	if err != nil {
		pkgLogger.Errorf("Unable to parse eventSch sch")
	}
	pkgLogger.Info(jsonProp)

	jsonSchemaProp := make(map[string]string)
	for k,v := range jsonProp {
		if strings.HasPrefix(k, "properties.") {
			keys := strings.Split(k, ".")
			if len(keys) != 2 {
				pkgLogger.Infof("multi level json found")
			}
			jsonSchemaProp[keys[1]] = v
		}
	}
	var jsonProp1 map[string]interface{}
	//jsonProp, ok := schema.(map[string]string)
	err = json.Unmarshal(schema, &jsonProp1)
	nested, err := unflatten(jsonProp1)
	if err!= nil {

	}
	pkgLogger.Infof("using inbuilt")
	pkgLogger.Info(nested)
	nested1,_:=json.Marshal(nested)
	pkgLogger.Info(string(nested1))

	// meta := eventTypes[0].WriteKey + ":" + eventTypes[0].EventType
	// generateJsonSchema(jsonSchemaProp, meta)
	eventTypesJSON, err := json.Marshal(eventTypes)
	if err != nil {
		http.Error(w, response.MakeResponse("Internal Error: Failed to Marshal event types"), 500)
		return
	}

	w.Write(eventTypesJSON)
}

type PropertyTypeT struct {
	Type []string `json:"type"`
}

type PropertiesT struct {
	Required []string //`json:"required"`
	Property map[string]PropertyTypeT `json:"properties"`
}

func generateJsonSchema(schemaProp map[string]string, meta string) {
	properties := PropertiesT{
		Property: make(map[string]PropertyTypeT),
	}
	required := make([]string,0)
	for k,v := range schemaProp {
		required = append(required, k)
		properties.Property[k] = PropertyTypeT {
			Type: []string{v},
		}
	}
	//pkgLogger.Info(string(json.Marshal(properties)))

	finalJson := make(map[string]interface{})
	finalJson["properties"] = properties.Property
	finalJson["required"] = required
	finalJson["$schema"] = "http://json-schema.org/draft-07/schema#"
	finalJson["additionalProperties"] = true
	finalJson["type"] = "object"
	finalJson["$id"] = "http://rudder.com/"+ meta
	finalJson["description"] = "Who bought what"

	json2, err := json.Marshal(finalJson)
	if err != nil {
		pkgLogger.Errorf("unable to form properties json:  %v", err)
	}

	pkgLogger.Info(string(json2))
}

//https://play.golang.org/p/4juOff38ea
//or use https://pkg.go.dev/github.com/wolfeidau/unflatten
//or use https://github.com/nqd/flat
func unflatten(flat map[string]interface{}) (map[string]interface{}, error) {
	unflat := map[string]interface{}{}

	for key, value := range flat {
		keyParts := strings.Split(key, ".")

		// Walk the keys until we get to a leaf node.
		m := unflat
		for i, k := range keyParts[:len(keyParts)-1] {
			v, exists := m[k]
			if !exists {
				newMap := map[string]interface{}{}
				m[k] = newMap
				m = newMap
				continue
			}

			innerMap, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("key=%v is not an object", strings.Join(keyParts[0:i+1], "."))
			}
			m = innerMap
		}

		leafKey := keyParts[len(keyParts)-1]
		if _, exists := m[leafKey]; exists {
			return nil, fmt.Errorf("key=%v already exists", key)
		}
		m[keyParts[len(keyParts)-1]] = value
	}

	return unflat, nil
}

func (manager *EventSchemaManagerT) GetEventVersions(w http.ResponseWriter, r *http.Request) {
	err := handleBasicAuth(r)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 400)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, response.MakeResponse("Only HTTP GET method is supported"), 400)
		return
	}

	eventIDs, ok := r.URL.Query()["EventID"]
	if !ok {
		http.Error(w, response.MakeResponse("Mandatory field: EventID missing"), 400)
		return
	}
	eventID := eventIDs[0]

	schemaVersions := manager.fetchSchemaVersionsByEventID(eventID)
	schemaVersionsJSON, err := json.Marshal(schemaVersions)
	if err != nil {
		http.Error(w, response.MakeResponse("Internal Error: Failed to Marshal event types"), 500)
		return
	}

	w.Write(schemaVersionsJSON)
}

//TODO: Complete this
func (manager *EventSchemaManagerT) GetKeyCounts(w http.ResponseWriter, r *http.Request) {
	err := handleBasicAuth(r)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 400)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, response.MakeResponse("Only HTTP GET method is supported"), 400)
		return
	}

	vars := mux.Vars(r)
	eventID, ok := vars["EventID"]
	if !ok {
		http.Error(w, response.MakeResponse("Mandatory field: EventID missing"), 400)
		return
	}

	keyCounts, err := manager.getKeyCounts(eventID)
	if err != nil {
		logID := uuid.NewV4().String()
		pkgLogger.Errorf("logID : %s, err: %s", logID, err.Error())
		http.Error(w, response.MakeResponse(fmt.Sprintf("Internal Error: An error has been logged with logID : %s", logID)), 500)
		return
	}
	keyCountsJSON, err := json.Marshal(keyCounts)
	if err != nil {
		logID := uuid.NewV4().String()
		pkgLogger.Errorf("logID : %s, err: %s", logID, err.Error())
		http.Error(w, response.MakeResponse(fmt.Sprintf("Interna Error: An error has been logged with logID : %s", logID)), 500)
		return
	}

	w.Write(keyCountsJSON)
}

func (manager *EventSchemaManagerT) getKeyCounts(eventID string) (keyCounts map[string]int64, err error) {

	schemaVersions := manager.fetchSchemaVersionsByEventID(eventID)

	keyCounts = make(map[string]int64)
	for _, sv := range schemaVersions {
		var schema map[string]string
		err = json.Unmarshal(sv.Schema, &schema)
		if err != nil {
			return
		}
		for k := range schema {
			_, ok := keyCounts[k]
			if !ok {
				keyCounts[k] = 0
			}
			keyCounts[k] = keyCounts[k] + sv.TotalCount
		}
	}
	return
}

func (manager *EventSchemaManagerT) GetEventModelMetadata(w http.ResponseWriter, r *http.Request) {
	err := handleBasicAuth(r)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 400)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, response.MakeResponse("Only HTTP GET method is supported"), 400)
		return
	}

	vars := mux.Vars(r)
	eventID, ok := vars["EventID"]
	if !ok {
		http.Error(w, response.MakeResponse("Mandatory field: VersionID missing"), 400)
		return
	}

	metadata, err := manager.fetchMetadataByEventModelID(eventID)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 400)
		return
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		http.Error(w, response.MakeResponse("Internal Error: Failed to Marshal metadata"), 500)
		return
	}

	w.Write(metadataJSON)

}

func (manager *EventSchemaManagerT) GetSchemaVersionMetadata(w http.ResponseWriter, r *http.Request) {
	err := handleBasicAuth(r)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 400)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, response.MakeResponse("Only HTTP GET method is supported"), 400)
		return
	}

	vars := mux.Vars(r)
	versionID, ok := vars["VersionID"]
	if !ok {
		http.Error(w, response.MakeResponse("Mandatory field: VersionID missing"), 400)
		return
	}

	metadata, err := manager.fetchMetadataByEventVersionID(versionID)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 400)
		return
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		http.Error(w, response.MakeResponse("Internal Error: Failed to Marshal metadata"), 500)
		return
	}

	w.Write(metadataJSON)
}

func (manager *EventSchemaManagerT) GetSchemaVersionMissingKeys(w http.ResponseWriter, r *http.Request) {
	err := handleBasicAuth(r)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 400)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, response.MakeResponse("Only HTTP GET method is supported"), 400)
		return
	}

	vars := mux.Vars(r)
	versionID, ok := vars["VersionID"]
	if !ok {
		http.Error(w, response.MakeResponse("Mandatory field: VersionID missing"), 400)
		return
	}

	schema, err := manager.fetchSchemaVersionByID(versionID)
	if err != nil {
		http.Error(w, response.MakeResponse(err.Error()), 500)
		return
	}

	eventModel, err := manager.fetchEventModelByID(schema.EventModelID)
	if err != nil {
		w.Write([]byte("[]"))
		return
	}

	schemaMap := make(map[string]string)
	masterSchemaMap := make(map[string]string)

	err = json.Unmarshal(schema.Schema, &schemaMap)
	if err != nil {
		logID := uuid.NewV4().String()
		pkgLogger.Errorf("logID : %s, err: %s", logID, err.Error())
		http.Error(w, response.MakeResponse(fmt.Sprintf("Internal Error: An error has been logged with logID : %s", logID)), 500)
		return
	}

	err = json.Unmarshal(eventModel.Schema, &masterSchemaMap)
	if err != nil {
		logID := uuid.NewV4().String()
		pkgLogger.Errorf("logID : %s, err: %s", logID, err.Error())
		http.Error(w, response.MakeResponse(fmt.Sprintf("Interna Error: An error has been logged with logID : %s", logID)), 500)
		return
	}

	missingKeys := make([]string, 0)

	for k := range masterSchemaMap {
		if _, ok := schemaMap[k]; !ok {
			missingKeys = append(missingKeys, k)
		}
	}

	missingKeyJSON, err := json.Marshal(missingKeys)
	if err != nil {
		http.Error(w, response.MakeResponse("Internal Error: Failed to Marshal metadata"), 500)
		return
	}

	w.Write(missingKeyJSON)
}

func (manager *EventSchemaManagerT) fetchEventModelsByWriteKey(writeKey string) []*EventModelT {
	var eventModelsSelectSQL string
	if writeKey == "" {
		eventModelsSelectSQL = fmt.Sprintf(`SELECT id, uuid, write_key, event_type, event_model_identifier, created_at, schema, total_count, last_seen FROM %s`, EVENT_MODELS_TABLE)
	} else {
		eventModelsSelectSQL = fmt.Sprintf(`SELECT id, uuid, write_key, event_type, event_model_identifier, created_at, schema, total_count, last_seen FROM %s WHERE write_key = '%s'`, EVENT_MODELS_TABLE, writeKey)
	}

	rows, err := manager.dbHandle.Query(eventModelsSelectSQL)
	assertError(err)
	defer rows.Close()

	eventModels := make([]*EventModelT, 0)

	for rows.Next() {
		var eventModel EventModelT
		err := rows.Scan(&eventModel.ID, &eventModel.UUID, &eventModel.WriteKey, &eventModel.EventType,
			&eventModel.EventIdentifier, &eventModel.CreatedAt, &eventModel.Schema, &eventModel.TotalCount, &eventModel.LastSeen)
		assertError(err)

		eventModels = append(eventModels, &eventModel)
	}

	return eventModels
}

func (manager *EventSchemaManagerT) fetchSchemaVersionsByEventID(eventID string) []*SchemaVersionT {
	schemaVersionsSelectSQL := fmt.Sprintf(`SELECT id, uuid, event_model_id, schema, first_seen, last_seen, total_count FROM %s WHERE event_model_id = '%s'`, SCHEMA_VERSIONS_TABLE, eventID)

	rows, err := manager.dbHandle.Query(schemaVersionsSelectSQL)
	assertError(err)
	defer rows.Close()

	schemaVersions := make([]*SchemaVersionT, 0)

	for rows.Next() {
		var schemaVersion SchemaVersionT
		err := rows.Scan(&schemaVersion.ID, &schemaVersion.UUID, &schemaVersion.EventModelID,
			&schemaVersion.Schema, &schemaVersion.FirstSeen, &schemaVersion.LastSeen, &schemaVersion.TotalCount)
		assertError(err)

		schemaVersions = append(schemaVersions, &schemaVersion)
	}

	return schemaVersions
}

func (manager *EventSchemaManagerT) fetchEventModelByID(id string) (*EventModelT, error) {
	eventModelsSelectSQL := fmt.Sprintf(`SELECT id, uuid, write_key, event_type, event_model_identifier, created_at, schema, total_count, last_seen FROM %s WHERE uuid = '%s'`, EVENT_MODELS_TABLE, id)

	rows, err := manager.dbHandle.Query(eventModelsSelectSQL)
	assertError(err)
	defer rows.Close()

	eventModels := make([]*EventModelT, 0)

	for rows.Next() {
		var eventModel EventModelT
		err := rows.Scan(&eventModel.ID, &eventModel.UUID, &eventModel.WriteKey, &eventModel.EventType,
			&eventModel.EventIdentifier, &eventModel.CreatedAt, &eventModel.Schema, &eventModel.TotalCount, &eventModel.LastSeen)
		assertError(err)

		eventModels = append(eventModels, &eventModel)
	}

	if len(eventModels) == 0 {
		err = fmt.Errorf("No eventModels found for given eventModelID : %s", id)
		return nil, err
	}

	if len(eventModels) > 1 {
		panic(fmt.Sprintf("More than one entry found for eventModelId : %s. Make sure a unique key constraint is present on uuid column", id))
	}

	return eventModels[0], nil
}

func (manager *EventSchemaManagerT) fetchSchemaVersionByID(id string) (*SchemaVersionT, error) {
	schemaVersionsSelectSQL := fmt.Sprintf(`SELECT id, uuid, event_model_id, schema, first_seen, last_seen, total_count FROM %s WHERE uuid = '%s'`, SCHEMA_VERSIONS_TABLE, id)

	rows, err := manager.dbHandle.Query(schemaVersionsSelectSQL)
	assertError(err)
	defer rows.Close()

	schemaVersions := make([]*SchemaVersionT, 0)

	for rows.Next() {
		var schemaVersion SchemaVersionT
		err := rows.Scan(&schemaVersion.ID, &schemaVersion.UUID, &schemaVersion.EventModelID, &schemaVersion.Schema, &schemaVersion.FirstSeen, &schemaVersion.LastSeen, &schemaVersion.TotalCount)
		assertError(err)

		schemaVersions = append(schemaVersions, &schemaVersion)
	}

	if len(schemaVersions) == 0 {
		err = fmt.Errorf("No SchemaVersion found for given VersionID : %s", id)
		return nil, err
	}

	if len(schemaVersions) > 1 {
		panic(fmt.Sprintf("More than one entry found for eventVersionID : %s. Make sure a unique key constraint is present on uuid column", id))
	}

	return schemaVersions[0], nil
}

func (manager *EventSchemaManagerT) fetchMetadataByEventVersionID(eventVersionID string) (metadata *MetaDataT, err error) {
	metadataSelectSQL := fmt.Sprintf(`SELECT metadata FROM %s WHERE uuid = '%s'`, SCHEMA_VERSIONS_TABLE, eventVersionID)

	rows, err := manager.dbHandle.Query(metadataSelectSQL)
	assertError(err)
	defer rows.Close()

	metadatas := make([]*MetaDataT, 0)

	for rows.Next() {
		var metadataRaw []byte
		err := rows.Scan(&metadataRaw)
		assertError(err)

		var metadata MetaDataT
		err = json.Unmarshal(metadataRaw, &metadata)
		assertError(err)
		metadatas = append(metadatas, &metadata)
	}

	if len(metadatas) > 1 {
		err = fmt.Errorf("More than one entry found for eventVersionID : %s. Make sure a unique key constraint is present on uuid column", eventVersionID)
		assertError(err)
	}

	if len(metadatas) == 0 {
		err = fmt.Errorf("No Metadata found for given VersionID : %s", eventVersionID)
		return nil, err
	}

	metadata = metadatas[0]
	return
}

func (manager *EventSchemaManagerT) fetchMetadataByEventModelID(eventModelID string) (metadata *MetaDataT, err error) {
	metadataSelectSQL := fmt.Sprintf(`SELECT metadata FROM %s WHERE uuid = '%s'`, EVENT_MODELS_TABLE, eventModelID)

	rows, err := manager.dbHandle.Query(metadataSelectSQL)
	assertError(err)
	defer rows.Close()

	metadatas := make([]*MetaDataT, 0)

	for rows.Next() {
		var metadataRaw []byte
		err := rows.Scan(&metadataRaw)
		assertError(err)

		var metadata MetaDataT
		err = json.Unmarshal(metadataRaw, &metadata)
		assertError(err)
		metadatas = append(metadatas, &metadata)
	}

	if len(metadatas) > 1 {
		err = fmt.Errorf("More than one entry found for eventVersionID : %s. Make sure a unique key constraint is present on uuid column", eventModelID)
		assertError(err)
	}

	if len(metadatas) == 0 {
		err = fmt.Errorf("No Metadata found for given VersionID : %s", eventModelID)
		return nil, err
	}

	metadata = metadatas[0]
	return
}
