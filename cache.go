package structwalker

import (
	"reflect"
	"sync"
)

type typeCache struct {
	m sync.Map // map[reflect.Type]*typeMeta
}

type typeMeta struct {
	Fields []fieldMeta
}

type fieldMeta struct {
	StructField reflect.StructField
	Tags        []Tag
	TagMap      map[string]Tag // key → Tag for fast lookup during dispatch
	IsStruct    bool
	IsPtr       bool
	IsSlice     bool
	IsMap       bool
	HandlerKeys []string
}

func newTypeCache() *typeCache {
	return &typeCache{}
}

func (c *typeCache) get(t reflect.Type, handlerKeys []string, tagParser func(string, string) Tag) *typeMeta {
	if meta, ok := c.m.Load(t); ok {
		return meta.(*typeMeta)
	}

	meta := buildTypeMeta(t, handlerKeys, tagParser)
	actual, _ := c.m.LoadOrStore(t, meta)
	return actual.(*typeMeta)
}

func buildTypeMeta(t reflect.Type, handlerKeys []string, tagParser func(string, string) Tag) *typeMeta {
	n := t.NumField()
	meta := &typeMeta{
		Fields: make([]fieldMeta, 0, n),
	}

	keySet := make(map[string]struct{}, len(handlerKeys))
	for _, k := range handlerKeys {
		keySet[k] = struct{}{}
	}

	for i := range n {
		sf := t.Field(i)
		var tags []Tag
		if tagParser != nil {
			for _, stdTag := range ParseAllTags(sf.Tag) {
				tags = append(tags, tagParser(stdTag.Key, stdTag.Raw))
			}
		} else {
			tags = ParseAllTags(sf.Tag)
		}

		ft := sf.Type
		isPtr := ft.Kind() == reflect.Ptr
		if isPtr {
			ft = ft.Elem()
		}

		fm := fieldMeta{
			StructField: sf,
			Tags:        tags,
			IsStruct:    ft.Kind() == reflect.Struct,
			IsPtr:       isPtr,
			IsSlice:     sf.Type.Kind() == reflect.Slice || sf.Type.Kind() == reflect.Array,
			IsMap:       sf.Type.Kind() == reflect.Map,
		}

		// Pre-compute tag map and matching handler keys
		if len(tags) > 0 {
			fm.TagMap = make(map[string]Tag, len(tags))
			for _, tag := range tags {
				fm.TagMap[tag.Key] = tag
				if _, ok := keySet[tag.Key]; ok {
					fm.HandlerKeys = append(fm.HandlerKeys, tag.Key)
				}
			}
		}

		meta.Fields = append(meta.Fields, fm)
	}

	return meta
}

func (c *typeCache) invalidate(t reflect.Type) {
	c.m.Delete(t)
}

func (c *typeCache) clear() {
	c.m.Range(func(key, _ any) bool {
		c.m.Delete(key)
		return true
	})
}
