package u

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//todo generic
// BSONDocValueForPath returns a value of the path. Returns defaultValue if not found or any error occured.
// {
//	 "list": [
// 		{
// 			"name": "foo"
// 		}
// 	 ]
// }
// EX: To get value of 'name'. Call BSONDocValueForPath(doc, nil, "list", 0, "name").
func BSONDocValueForPath(doc bson.D, defaultValue interface{}, path ...interface{}) interface{} {
	if doc == nil {
		return defaultValue
	}
	var currVal interface{} = doc
PathLoop:
	for _, component := range path {
		k, ok := component.(string)
		if ok {
			// component is a string. component must be a key.
			// currVal must be a bson.D
			d, ok := currVal.(bson.D)
			if !ok {
				// currVal is not a bson.D. unable to continue parsing.
				return defaultValue
			}
			// find a value match k
			for _, e := range d {
				if k == e.Key {
					currVal = e.Value
					continue PathLoop
				}
			}
			return defaultValue
		}

		// if component is not a key. component must be an index of a slice.
		i, ok := component.(int)
		if !ok {
			// component is neither string nor int. unable to continue.
			return defaultValue
		}
		slc, ok := currVal.(bson.A)
		if !ok {
			// slc is not a slice. unable to continue.
			return defaultValue
		}
		if i >= 0 && i < len(slc) {
			currVal = slc[i]
		} else {
			// out of bounds.
			return defaultValue
		}
	}
	return currVal
}

func BSONTS(tsSec int64) primitive.Timestamp {
	return primitive.Timestamp{T: uint32(tsSec)}
}
