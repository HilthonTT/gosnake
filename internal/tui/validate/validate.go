package validate

import "fmt"

type ErrorHandler func(str string) error

func NotEmpty(name string) ErrorHandler {
	return func(str string) error {
		if len(str) == 0 {
			return fmt.Errorf("%s cannot be empty", name)
		}
		return nil
	}
}
