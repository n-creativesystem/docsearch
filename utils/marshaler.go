package utils

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/registry"
)

func init() {
	registry.RegisterType("protobuf.LivenessCheckResponse", reflect.TypeOf(protobuf.LivenessCheckResponse{}))
	registry.RegisterType("protobuf.ReadinessCheckResponse", reflect.TypeOf(protobuf.ReadinessCheckResponse{}))
	registry.RegisterType("protobuf.Metadata", reflect.TypeOf(protobuf.Metadata{}))
	registry.RegisterType("protobuf.Node", reflect.TypeOf(protobuf.Node{}))
	registry.RegisterType("protobuf.Cluster", reflect.TypeOf(protobuf.Cluster{}))
	registry.RegisterType("protobuf.JoinRequest", reflect.TypeOf(protobuf.JoinRequest{}))
	registry.RegisterType("protobuf.LeaveRequest", reflect.TypeOf(protobuf.LeaveRequest{}))
	registry.RegisterType("protobuf.NodeResponse", reflect.TypeOf(protobuf.NodeResponse{}))
	registry.RegisterType("protobuf.ClusterResponse", reflect.TypeOf(protobuf.ClusterResponse{}))
	registry.RegisterType("protobuf.DeleteDocument", reflect.TypeOf(protobuf.DeleteDocument{}))
	registry.RegisterType("protobuf.Documents", reflect.TypeOf(protobuf.Documents{}))
	registry.RegisterType("protobuf.DeleteDocuments", reflect.TypeOf(protobuf.DeleteDocuments{}))
	registry.RegisterType("protobuf.SearchRequest", reflect.TypeOf(protobuf.SearchRequest{}))
	registry.RegisterType("protobuf.SearchResponse", reflect.TypeOf(protobuf.SearchResponse{}))
	registry.RegisterType("protobuf.UserDictionaryRecords", reflect.TypeOf(protobuf.UserDictionaryRecords{}))
	registry.RegisterType("protobuf.DeleteDictionaryRequest", reflect.TypeOf(protobuf.DeleteDictionaryRequest{}))
	registry.RegisterType("protobuf.DeleteDictionaries", reflect.TypeOf(protobuf.DeleteDictionaries{}))
	registry.RegisterType("protobuf.DictionaryResponse", reflect.TypeOf(protobuf.DictionaryResponse{}))
	registry.RegisterType("protobuf.GetMetadataRequest", reflect.TypeOf(protobuf.GetMetadataRequest{}))
	registry.RegisterType("protobuf.SetMetadataRequest", reflect.TypeOf(protobuf.SetMetadataRequest{}))
	registry.RegisterType("protobuf.DeleteMetadataRequest", reflect.TypeOf(protobuf.DeleteMetadataRequest{}))
	registry.RegisterType("protobuf.Event", reflect.TypeOf(protobuf.Event{}))
	registry.RegisterType("protobuf.WatchResponse", reflect.TypeOf(protobuf.WatchResponse{}))
	registry.RegisterType("protobuf.MetricsResponse", reflect.TypeOf(protobuf.MetricsResponse{}))
	registry.RegisterType("protobuf.Document", reflect.TypeOf(protobuf.Document{}))
	registry.RegisterType("map[string]interface {}", reflect.TypeOf((map[string]interface{})(nil)))
	registry.RegisterType("string", reflect.TypeOf(""))
	registry.RegisterType("float64", reflect.TypeOf(float64(0)))
	registry.RegisterType("time.Time", reflect.TypeOf(time.Time{}))
	registry.RegisterType("[]uint8", reflect.TypeOf(([]byte)(nil)))
	registry.RegisterType("interface {}", reflect.TypeOf((interface{})(nil)))
}

func MarshalAny(message *any.Any) (interface{}, error) {
	if message == nil {
		return nil, nil
	}

	typeUrl := message.TypeUrl
	value := message.Value

	instance := registry.TypeInstanceByName(typeUrl)

	if err := json.Unmarshal(value, instance); err != nil {
		return nil, err
	} else {
		return instance, nil
	}

}

func UnmarshalAny(instance interface{}, message *any.Any) error {
	if instance == nil {
		return nil
	}
	value, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	message.TypeUrl = registry.TypeNameByInstance(instance)
	message.Value = value
	return nil
}
