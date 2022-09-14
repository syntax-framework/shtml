package cmn

import "encoding/json"

// JSON https://www.json.org/json-en.html
type JSON map[string]interface{}

func JSONParse(data []byte) (*JSON, error) {
	var obj = &JSON{}
	err := json.Unmarshal(data, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (j *JSON) Encode() ([]byte, error) {
	return json.Marshal(j.Get())
}

func (j *JSON) ToStruct() {

}

func (j *JSON) ToStructArray() {

}

func (j *JSON) Get() map[string]interface{} {
	o := *j
	if o == nil {
		return map[string]interface{}{}
	}
	return o
}

// Has determine if the JSON contains a specific key.
func (j *JSON) Has(key string) (exists bool) {
	_, exists = j.Get()[key]
	return
}

func (j *JSON) String(key string) string {
	return ""
}

func (j *JSON) Number(key string) bool {
	return false
}

func (j *JSON) Bool(key string) bool {
	return false
}

func (j *JSON) Object(key string) *JSON {
	return nil
}

func (j *JSON) Array(key string) []*JSON {
	return nil
}

func (j *JSON) ArrayString(key string) []string {
	return nil
}

func (j *JSON) ArrayNumber(key string) bool {
	return false
}

func (j *JSON) ArrayBool(key string) []bool {
	return nil
}
