package ptr

// ToValue converts a pointer to a value
// It safely handles nil pointers by providing a zero value
func ToValue[T any](ptr *T) T {
	var zero T
	if ptr == nil {
		return zero
	}
	return *ptr
}

// ToPtr converts a value to a pointer
func ToPtr[T any](value T) *T {
	return &value
}
