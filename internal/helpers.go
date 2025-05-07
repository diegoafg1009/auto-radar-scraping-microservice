package internal

import "strconv"

func StringToUint32(str string) (uint32, error) {
	num, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(num), nil
}
