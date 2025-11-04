package comparer

import "github.com/pkg/errors"

// FormatErrorWithQuery wraps an error with the SQL query that caused it.
func FormatErrorWithQuery(err error, query string) error {
	if err == nil {
		return nil
	}
	return errors.Wrapf(err, "query error\nQuery: %s", query)
}

// ConvertYesNo converts MySQL/PostgreSQL YES/NO strings to bool.
func ConvertYesNo(s string) (bool, error) {
	switch s {
	case "YES", "yes", "Y", "y", "1", "true", "t":
		return true, nil
	case "NO", "no", "N", "n", "0", "false", "f":
		return false, nil
	default:
		return false, errors.Errorf("unrecognized yes/no value: %s", s)
	}
}
