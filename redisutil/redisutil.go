package redisutil

import "fmt"

func Strings(i interface{}, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}

	vv, ok := i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type: %T expected to be a []interface{}", i)
	}

	ss := make([]string, 0, len(vv))
	for _, v := range vv {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type: %T expected to be a string", v)
		}
		ss = append(ss, s)
	}

	return ss, nil
}
