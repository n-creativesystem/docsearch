package server

import (
	"time"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/utils"
	"google.golang.org/protobuf/proto"
)

// Upload is ドキュメンの追加・更新
func (s *RaftServer) Upload(req *protobuf.Documents) error {
	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(req, dataAny); err != nil {
		return err
	}
	event := &protobuf.Event{
		Type: protobuf.Event_Upload,
		Data: dataAny,
	}
	msg, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	timeout := 60 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		return future.Error()
	}
	return nil
}

// Search is ドキュメントの検索
func (s *RaftServer) Search(req *protobuf.SearchRequest) (*protobuf.SearchResponse, error) {
	buf, err := s.fsm.Search(req)
	if err != nil {
		return nil, err
	}
	res := &protobuf.SearchResponse{
		Result: buf,
	}
	return res, nil
}

// AddUserDictionary is ユーザー辞書の追加
func (s *RaftServer) AddUserDictionary(req *protobuf.UserDictionaryRecords) (*protobuf.DictionaryResponse, error) {
	resp := &protobuf.DictionaryResponse{
		Results: false,
	}
	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(req, dataAny); err != nil {
		return resp, err
	}
	event := &protobuf.Event{
		Type: protobuf.Event_Dictionary,
		Data: dataAny,
	}
	msg, err := proto.Marshal(event)
	if err != nil {
		return resp, err
	}
	timeout := 60 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		return resp, err
	}
	resp.Results = true
	return resp, nil
}

func (s *RaftServer) RemoveDictionary(req *protobuf.DeleteDictionaryRequest) (*protobuf.DictionaryResponse, error) {
	resp := &protobuf.DictionaryResponse{
		Results: false,
	}
	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(req, dataAny); err != nil {
		return resp, err
	}
	event := &protobuf.Event{
		Type: protobuf.Event_RemoveDictionary,
		Data: dataAny,
	}
	msg, err := proto.Marshal(event)
	if err != nil {
		return resp, err
	}
	timeout := 60 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		return resp, err
	}
	resp.Results = true
	return resp, nil
}

// BulkDelete is ドキュメントの一括削除
func (s *RaftServer) BulkDelete(req *protobuf.DeleteDocuments) error {
	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(req, dataAny); err != nil {
		return err
	}
	event := &protobuf.Event{
		Type: protobuf.Event_Delete,
		Data: dataAny,
	}
	msg, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	timeout := 60 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		return future.Error()
	}
	return nil
}
