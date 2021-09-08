package helper

import (
	"strings"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/numeric/geo"
	"github.com/n-creativesystem/docsearch/analyzer"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/protobuf"
)

func isBoostDefault(boost float64) bool {
	return boost != 1.0
}

func MatchAllQuery(query *protobuf.MatchAllQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewMatchAllQuery()
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func MatchQuery(query *protobuf.MatchQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewMatchQuery(query.Match).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	if query.AnalyzerName != "" {
		if ana := analyzer.GetAnalyzer(query.AnalyzerName); ana != nil {
			q.SetAnalyzer(ana)
		}
	}
	switch query.Operator {
	case protobuf.MatchQuery_Or:
		q.SetOperator(bluge.MatchQueryOperatorOr)
	case protobuf.MatchQuery_And:
		q.SetOperator(bluge.MatchQueryOperatorAnd)
	}
	return q, nil
}

func MatchNoneQuery(query *protobuf.MatchNoneQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewMatchNoneQuery()
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func MatchPhraseQuery(query *protobuf.MatchPhraseQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewMatchPhraseQuery(query.MatchPhrase).SetField(query.Field)
	if query.AnalyzerName != "" {
		if ana := analyzer.GetAnalyzer(query.AnalyzerName); ana != nil {
			q.SetAnalyzer(ana)
		}
	}
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	q.SetSlop(int(query.GetSlop()))
	return q, nil
}

func MultiPhraseQuery(query *protobuf.MultiPhraseQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewMultiPhraseQuery([][]string{query.Terms}).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	q.SetSlop(int(query.Slop))
	return q, nil
}

func PrefixQuery(query *protobuf.PrefixQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewPrefixQuery(query.Prefix).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func RegexpQuery(query *protobuf.RegexpQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewRegexpQuery(query.Regexp).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func TermQuery(query *protobuf.TermQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewTermQuery(query.Term).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func TermRangeQuery(query *protobuf.TermRangeQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewTermRangeQuery(query.Min, query.Max).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func map2QueriesQuery(queryMap map[string]*protobuf.Query) ([]bluge.Query, error) {
	queries := make([]bluge.Query, 0, len(queryMap))
	for field, value := range queryMap {
		if q, err := GetQuery(field, value); err != nil {
			return nil, err
		} else {
			queries = append(queries, q)
		}
	}
	return queries, nil
}

func BooleanQuery(query *protobuf.BooleanQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	musts := make([]bluge.Query, 0, len(query.Musts))
	if q, err := map2QueriesQuery(query.Musts); err != nil {
		return nil, err
	} else {
		musts = append(musts, q...)
	}
	mustNots := make([]bluge.Query, 0, len(query.MustNots))
	if q, err := map2QueriesQuery(query.MustNots); err != nil {
		return nil, err
	} else {
		mustNots = append(mustNots, q...)
	}
	shoulds := make([]bluge.Query, 0, len(query.Shoulds))
	if q, err := map2QueriesQuery(query.Shoulds); err != nil {
		return nil, err
	} else {
		shoulds = append(shoulds, q...)
	}

	q := bluge.NewBooleanQuery().AddMust(musts...).AddMustNot(mustNots...).AddShould(shoulds...)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	if query.MinShould != 0 {
		q.SetMinShould(int(query.MinShould))
	}
	return q, nil
}

func DateRangeQuery(query *protobuf.DateRangeQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	start, err := ParseISO8601(query.Start)
	if err != nil {
		return nil, err
	}
	end, err := ParseISO8601(query.End)
	if err != nil {
		return nil, err
	}
	q := bluge.NewDateRangeQuery(start, end).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func GeoBoundingBoxQuery(query *protobuf.GeoBoundingBoxQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewGeoBoundingBoxQuery(query.TopLeftLon, query.TopLeftLat, query.BottomRightLon, query.BottomRightLat).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func GeoDistanceQuery(query *protobuf.GeoDistanceQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewGeoDistanceQuery(query.Point.Lon, query.Point.Lat, query.Distance).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func GeoBoundingPolygonQuery(query *protobuf.GeoBoundingPolygonQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	points := make([]geo.Point, 0, len(query.Points))
	for _, p := range query.Points {
		point := geo.Point{
			Lon: p.Lon,
			Lat: p.Lat,
		}
		points = append(points, point)
	}
	q := bluge.NewGeoBoundingPolygonQuery(points).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func NumericRangeQuery(query *protobuf.NumericRangeQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewNumericRangeQuery(query.Min, query.Max).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func WildcardQuery(query *protobuf.WildcardQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewWildcardQuery(query.Wildcard).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	return q, nil
}

func FuzzyQuery(query *protobuf.FuzzyQuery) (bluge.Query, error) {
	if query == nil {
		return nil, errors.Nil
	}
	q := bluge.NewFuzzyQuery(query.Term).SetField(query.Field)
	if isBoostDefault(query.Boost) {
		q.SetBoost(query.Boost)
	}
	q.SetFuzziness(int(query.Fuzziness))
	q.SetPrefix(int(query.Prefix))
	return q, nil
}

func GetQuery(field string, query *protobuf.Query) (bluge.Query, error) {
	switch strings.ToLower(field) {
	case "match_all":
		return MatchAllQuery(query.GetMatchAll())
	case "match_query":
		return MatchQuery(query.GetMatchQuery())
	case "match_none":
		return MatchNoneQuery(query.GetMatchNone())
	case "match_phrase":
		return MatchPhraseQuery(query.GetMatchPhrase())
	case "multi_phrase":
		return MultiPhraseQuery(query.GetMultiPhrase())
	case "prefix":
		return PrefixQuery(query.GetPrefix())
	case "regexp":
		return RegexpQuery(query.GetRegexp())
	case "term":
		return TermQuery(query.GetTerm())
	case "term_range":
		return TermRangeQuery(query.GetTermRange())
	case "bool":
		return BooleanQuery(query.GetBool())
	case "date_range":
		return DateRangeQuery(query.GetDateRange())
	case "geo_bounding_box":
		return GeoBoundingBoxQuery(query.GetGeoBoundingBox())
	case "geo_distance":
		return GeoDistanceQuery(query.GetGeoDistance())
	case "geo_bounding_polygon":
		return GeoBoundingPolygonQuery(query.GetGeoBoundingPolygon())
	case "numeric_range":
		return NumericRangeQuery(query.GetNumericRange())
	case "wildcard":
		return WildcardQuery(query.GetWildcard())
	case "fuzzy":
		return FuzzyQuery(query.GetFuzzy())
	default:
		return nil, errors.New("No support query: %s", field)
	}
}
