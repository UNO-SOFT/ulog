// Copyright 2020 Tamás Gulácsi.
// Copyright 2019 The Antilog Authors.
//
// SPDX-License-Identifier: MIT

package ulog

// Field type for all inputs
type Field interface{}

// EncodedField type for storing fields in after conversion to JSON
type encodedField [2]string

// Key of the encoded field
func (f encodedField) Key() string {
	return f[0]
}

// Value of the encoded field
func (f encodedField) Value() string {
	return f[1]
}

// encodedFields is a list of encoded fields
type encodedFields []encodedField

// Add and encode fields.
func (eF *encodedFields) AppendFields(fields []Field) *encodedFields {
	for ix := 0; ix < len(fields); ix += 2 {
		rawKey := fields[ix]
		rawValue := fields[ix+1]

		keyString, ok := rawKey.(string)
		if !ok {
			continue
		}

		key := toJSON(keyString)
		value := toJSON(rawValue)

		if i := eF.Index(key); i >= 0 {
			(*eF)[i][1] = value
			continue
		}

		*eF = append(*eF, encodedField{key, value})
	}
	return eF
}

// AppendUnique encoded field if the key is not already set
func (eF *encodedFields) AppendEncoded(fields encodedFields) *encodedFields {
	if eF == nil {
		return eF
	}
	for _, f := range fields {
		if i := eF.Index(f.Key()); i >= 0 {
			(*eF)[i][1] = f.Value()
		} else {
			*eF = append(*eF, f)
		}
	}
	return eF
}

func (eF *encodedFields) Index(key string) int {
	if eF == nil {
		return -1
	}
	for i, v := range *eF {
		if v.Key() == key {
			return i
		}
	}
	return -1
}
