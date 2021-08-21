package server

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

func TestRaftServer(t *testing.T) {
	log := logrus.New()
	id := "node1"
	data := &protobuf.GetMetadataRequest{
		Id: id,
	}

	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(data, dataAny); err != nil {
		log.WithField("id", id).Error(err)
	}

	event := &protobuf.Event{
		Type: protobuf.Event_GetNode,
		Data: dataAny,
	}

	msg, err := proto.Marshal(event)
	if err != nil {
		log.WithField("id", id).Error(err)
	}
	log.Infof("%s", string(msg))

	var event2 protobuf.Event
	if err := proto.Unmarshal(msg, &event2); err != nil {
		log.Error(err)
	}
	if data, err := convertData(&event2); err != nil {
		log.Error(err)
	} else {
		log.Infof("%s:%s", event2.Type, reflect.TypeOf(data))
	}
}

// convertData is raft コマンドの生成
func convertData(event *protobuf.Event) (interface{}, error) {
	data, err := utils.MarshalAny(event.Data)
	if err != nil {
		return nil, err
	}
	if data == nil {
		err = errors.Nil
		return nil, err
	}
	return data, nil
}
