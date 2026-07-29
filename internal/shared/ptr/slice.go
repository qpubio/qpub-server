package ptr

// PtrSliceToValueSlice converts a slice of pointers to a slice of values
// It safely handles nil pointers by providing zero values
func PtrSliceToValueSlice[T any](ptrSlice []*T) []T {
	if ptrSlice == nil {
		return nil
	}

	result := make([]T, len(ptrSlice))
	for i, ptr := range ptrSlice {
		if ptr != nil {
			result[i] = *ptr
		}
	}
	return result
}
