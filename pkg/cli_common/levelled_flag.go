package clicommon

import "strconv"

type LevelledFlag int

func (f *LevelledFlag) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		l, intErr := strconv.ParseInt(s, 10, 64)
		if intErr != nil {
			return err
		}
		*f = LevelledFlag(l)
		return nil
	}
	if v {
		*f++
	} else if *f > 0 {
		*f--
	}
	return nil
}

func (f *LevelledFlag) Type() string {
	return "levelled_flag"
}

func (f *LevelledFlag) String() string {
	return strconv.FormatInt(int64(*f), 10)
}
