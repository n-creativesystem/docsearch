package fsm

import (
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/blugelabs/bluge"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/storage"
	"github.com/n-creativesystem/docsearch/utils"
	"github.com/sirupsen/logrus"
)

type FSM struct {
	logger     *logrus.Logger
	index      *storage.Index
	metadata   map[string]*protobuf.Metadata
	nodesMutex sync.RWMutex

	ApplyCh chan *protobuf.Event
}

var _ raft.FSM = (*FSM)(nil)

func NewRaftFSM(path string, logger *logrus.Logger) (*FSM, error) {
	index, err := storage.New(path, logger)
	if err != nil {
		return nil, err
	}
	return &FSM{
		logger:   logger,
		index:    index,
		metadata: make(map[string]*protobuf.Metadata, 0),
		ApplyCh:  make(chan *protobuf.Event, 1024),
	}, nil
}

func (f *FSM) Apply(log *raft.Log) interface{} {
	var event protobuf.Event
	err := proto.Unmarshal(log.Data, &event)
	if err != nil {
		f.logger.Error(err)
		return err
	}
	data, err := f.convertData(&event)
	if err != nil {
		return NewApplyErr(err)
	}
	switch event.Type {
	case protobuf.Event_Join:
		req := data.(*protobuf.SetMetadataRequest)
		if err := f.setMetadata(req.Id, req.Metadata); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event
		return &ApplyResponse{}
	case protobuf.Event_Leave:
		req := data.(*protobuf.DeleteMetadataRequest)
		if err := f.deleteMetadata(req.Id); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event
		return &ApplyResponse{}
	case protobuf.Event_Delete:
		req := data.(*protobuf.DeleteDocuments).Requests
		var ids = make([]string, len(req))
		for i, r := range req {
			ids[i] = r.Id
		}
		if err := f.bulkDelete(ids); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event

		return &ApplyResponse{}
	case protobuf.Event_Upload:
		req := data.(*protobuf.Documents)
		var docs []*bluge.Document
		for _, r := range req.Requests {
			var field *bluge.TermField
			doc := bluge.NewDocument(r.Id)
			for k, v := range r.Fields {
				data, err := utils.MarshalAny(v.Body)
				if err != nil {
					f.logger.Warnf("data convert error: %s", err)
					continue
				}
				analyzer := storage.GetAnalyzer(v.AnalyzerName)
				switch v.Type {
				case protobuf.Field_Text:
					body, ok := data.(*string)
					if ok {
						field = bluge.NewTextField(k, *body)
					} else {
						continue
					}
				case protobuf.Field_TextBytes:
					body, ok := data.([]byte)
					if ok {
						field = bluge.NewTextFieldBytes(k, body)
					} else {
						continue
					}
				case protobuf.Field_Numeric:
					body, ok := data.(*float64)
					if ok {
						field = bluge.NewNumericField(k, *body)
					} else {
						continue
					}
				case protobuf.Field_GeoPoint:
					body, ok := data.(*string)
					if ok {
						geoPoint := strings.Split(*body, ",")
						if len(geoPoint) == 2 {
							lon, err := strconv.ParseFloat(geoPoint[0], 64)
							if err != nil {
								f.logger.Warnf("lon")
								continue
							}
							lat, err := strconv.ParseFloat(geoPoint[1], 64)
							if err != nil {
								continue
							}
							field = bluge.NewGeoPointField(k, lon, lat)
						} else {
							continue
						}
					}
				}
				if field != nil {
					if analyzer != nil {
						field.WithAnalyzer(analyzer)
					}
					doc.AddField(field.StoreValue().HighlightMatches())
				} else {
					continue
				}
			}
			docs = append(docs, doc)
		}
		if err := f.update(docs); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event

		return &ApplyResponse{}
	}
	return nil
}

func (f *FSM) Search(req bluge.SearchRequest) ([]string, error) {
	return f.index.Search(req)
}

func (f *FSM) Close() error {
	return f.index.Close()
}

func (f *FSM) GetMetadata(id string) *protobuf.Metadata {
	if metadata, exists := f.metadata[id]; exists {
		return metadata
	} else {
		f.logger.Debugf("metadata not found: %s", id)
		return nil
	}
}

func (f *FSM) update(docs []*bluge.Document) error {
	return f.index.Update(docs)
}

func (f *FSM) delete(id string) error {
	return f.index.Delete(id)
}

func (f *FSM) bulkDelete(ids []string) error {
	return f.index.BulkDelete(ids)
}

func (f *FSM) convertData(event *protobuf.Event) (interface{}, error) {
	data, err := utils.MarshalAny(event.Data)
	if err != nil {
		f.logger.Errorf("failed to marshal event data to set request: %s", event.Type.String())
		return nil, err
	}
	if data == nil {
		err = errors.Nil
		f.logger.Errorf("request is nil: %s", event.Type.String())
		return nil, err
	}
	return data, nil
}

func (f *FSM) setMetadata(id string, metadata *protobuf.Metadata) error {
	f.nodesMutex.Lock()
	defer f.nodesMutex.Unlock()

	f.metadata[id] = metadata

	return nil
}

func (f *FSM) deleteMetadata(id string) error {
	f.nodesMutex.Lock()
	defer f.nodesMutex.Unlock()

	if _, exists := f.metadata[id]; exists {
		delete(f.metadata, id)
	}

	return nil
}

func (f *FSM) join() {

}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return nil, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	return nil
}
