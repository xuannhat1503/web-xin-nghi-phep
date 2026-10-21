// Package dbtypes holds small SQL value types used by generated store code.
package dbtypes

import (
	"database/sql/driver"
	"fmt"
)

// JSON is a nullable JSON column. A nil/empty value is stored as SQL NULL and marshals as JSON null.
type JSON []byte

func (j *JSON) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*j = nil
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = JSON(v)
	default:
		return fmt.Errorf("dbtypes.JSON: cannot scan %T", src)
	}
	return nil
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}
