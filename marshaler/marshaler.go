package marshaler

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/utils"
	"google.golang.org/protobuf/types/known/anypb"
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
	switch v.(type) {
	case *protobuf.SearchRequest:
		var searchResult map[string]interface{}
		if err := json.Unmarshal(v.(*protobuf.SearchResponse).Result, &searchResult); err != nil {
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

type documentField struct {
	Body         interface{} `json:"body"`
	AnalyzerName string      `json:"analyzer_name"`
	Type         string      `json:"type"`
}

func (field *documentField) ToProtobufField() (*protobuf.Field, error) {
	typ, ok := protobuf.Field_Type_value[field.Type]
	if !ok {
		typ = int32(protobuf.Field_Text)
	}
	var dataAny anypb.Any
	switch value := field.Body.(type) {
	case string:
		if err := utils.UnmarshalAny(&value, &dataAny); err != nil {
			return nil, err
		}
	case float32:
		if err := utils.UnmarshalAny(&value, &dataAny); err != nil {
			return nil, err
		}
	case float64:
		if err := utils.UnmarshalAny(&value, &dataAny); err != nil {
			return nil, err
		}
	case int32:
		if err := utils.UnmarshalAny(&value, &dataAny); err != nil {
			return nil, err
		}
	case int64:
		if err := utils.UnmarshalAny(&value, &dataAny); err != nil {
			return nil, err
		}
	case int:
		if err := utils.UnmarshalAny(&value, &dataAny); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("No support type: %s", reflect.TypeOf(field.Body).String())
	}
	return &protobuf.Field{
		AnalyzerName: field.AnalyzerName,
		Type:         protobuf.Field_Type(typ),
		Body:         &dataAny,
	}, nil
}

type document struct {
	Id     string                    `json:"id"`
	Fields map[string]*documentField `json:"fields"`
}
type documents struct {
	Requests []*document `json:"requests"`
}

func (m *GatewayMarshaler) Unmarshal(data []byte, v interface{}) error {
	switch value := v.(type) {
	case *protobuf.Document:
		var doc document
		if err := json.Unmarshal(data, &doc); err != nil {
			return err
		}
		value.Id = doc.Id
		value.Fields = make(map[string]*protobuf.Field, len(doc.Fields))
		for key, field := range doc.Fields {
			bufField, err := field.ToProtobufField()
			if err != nil {
				return err
			}
			value.Fields[key] = bufField
		}
		return nil
	case *protobuf.Documents:
		var docs documents
		if err := json.Unmarshal(data, &docs); err != nil {
			return err
		}
		value.Requests = make([]*protobuf.Document, len(docs.Requests))
		for idx, doc := range docs.Requests {
			doc := doc
			req := &protobuf.Document{
				Id: doc.Id,
			}
			req.Fields = make(map[string]*protobuf.Field, len(doc.Fields))
			for key, field := range doc.Fields {
				bufField, err := field.ToProtobufField()
				if err != nil {
					return err
				}
				req.Fields[key] = bufField
			}
			value.Requests[idx] = req
		}
		return nil
	case *protobuf.SearchRequest:
		type searchRequestBody struct {
			Query        string   `json:"query"`
			AnalyzerName string   `json:"analyzer_name"`
			Fields       []string `json:"fields"`
		}
		/* {
			"search_request": {
			"query": ""
			"analyzer_name": "",
			"fields": [],
			}
		}
		*/
		type request struct {
			SearchRequest *searchRequestBody `json:"search_request"`
		}
		var m request
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		ok := m.SearchRequest != nil
		if !ok {
			return errors.Nil
		}
		value.Fields = append(v.(*protobuf.SearchRequest).Fields, m.SearchRequest.Fields...)
		value.AnalyzerName = m.SearchRequest.AnalyzerName
		value.Query = m.SearchRequest.Query
		return nil
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
