package marshaler

import (
	"encoding/json"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/n-creativesystem/docsearch/protobuf"
)

var (
	DefaultContentType = "application/json"
)

type GatewayMarshaler struct{}

var _ runtime.Marshaler = (*GatewayMarshaler)(nil)

func (*GatewayMarshaler) ContentType(v interface{}) string {
	return DefaultContentType
}

func (m *GatewayMarshaler) Marshal(v interface{}) ([]byte, error) {
	switch value := v.(type) {
	case *protobuf.SearchResponse:
		var searchResult interface{}
		if err := json.Unmarshal(value.Result, &searchResult); err != nil {
			return nil, err
		}
		resp := map[string]interface{}{
			"search_result": searchResult,
		}
		if value, err := json.Marshal(resp); err == nil {
			return value, nil
		} else {
			return nil, err
		}
	default:
		return json.Marshal(v)
	}
}

func (m *GatewayMarshaler) Unmarshal(data []byte, v interface{}) error {
	switch value := v.(type) {
	case *protobuf.Document:
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		if id, ok := m["id"].(string); ok {
			value.Id = id
		}
		if f, ok := m["fields"].(map[string]interface{}); ok {
			fieldsBytes, err := json.Marshal(f)
			if err != nil {
				return err
			}
			value.Fields = fieldsBytes
		}
		return nil
	case *protobuf.Documents:
		docs := []map[string]interface{}{}
		if err := json.Unmarshal(data, &docs); err != nil {
			return err
		}
		value.Requests = make([]*protobuf.Document, len(docs))
		for idx, d := range docs {
			docBytes, err := json.Marshal(&d)
			if err != nil {
				return err
			}
			doc := &protobuf.Document{}
			if err := m.Unmarshal(docBytes, doc); err != nil {
				return err
			}
			value.Requests[idx] = doc
		}
		return nil
	// case *protobuf.SearchRequest:
	// 	type searchRequestBody struct {
	// 		Query        string   `json:"query"`
	// 		AnalyzerName string   `json:"analyzer_name"`
	// 		Fields       []string `json:"fields"`
	// 	}
	// 	type request struct {
	// 		SearchRequest *searchRequestBody `json:"search_request"`
	// 	}
	// 	var m request
	// 	if err := json.Unmarshal(data, &m); err != nil {
	// 		return err
	// 	}
	// 	ok := m.SearchRequest != nil
	// 	if !ok {
	// 		return errors.Nil
	// 	}
	// 	value.Fields = append(v.(*protobuf.SearchRequest).Fields, m.SearchRequest.Fields...)
	// 	value.AnalyzerName = m.SearchRequest.AnalyzerName
	// 	value.Query = m.SearchRequest.Query
	// 	return nil
	default:
		return json.Unmarshal(data, v)
	}
}

func (m *GatewayMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(
		func(v interface{}) error {
			buffer, err := io.ReadAll(r)
			if err != nil {
				return err
			}

			return m.Unmarshal(buffer, v)
		},
	)
}

func (m *GatewayMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return json.NewEncoder(w)
}

func (m *GatewayMarshaler) Delimiter() []byte {
	return []byte("\n")
}
