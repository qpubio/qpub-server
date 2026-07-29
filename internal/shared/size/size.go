package size

import (
	"fmt"
	"strconv"
	"strings"
)

// Unit represents a size unit
type Unit string

const (
	// Units
	Byte Unit = "B"
	KiB  Unit = "KiB"
	MiB  Unit = "MiB"
	GiB  Unit = "GiB"
	TiB  Unit = "TiB"
	PiB  Unit = "PiB"

	// Conversion factors
	bytesInKiB = 1024
	bytesInMiB = 1024 * 1024
	bytesInGiB = 1024 * 1024 * 1024
	bytesInTiB = 1024 * 1024 * 1024 * 1024
	bytesInPiB = 1024 * 1024 * 1024 * 1024 * 1024
)

// Size represents a size with a value and unit
type Size struct {
	Value float64
	Unit  Unit
}

// String returns the string representation of the size
func (s Size) String() string {
	// For values less than 10, use one decimal place
	if s.Value < 10 && s.Value != float64(int64(s.Value)) {
		return fmt.Sprintf("%.1f %s", s.Value, s.Unit)
	}

	// For larger values, use integer representation
	return fmt.Sprintf("%d %s", int64(s.Value), s.Unit)
}

// ToBytes converts the size to bytes
func (s Size) ToBytes() int64 {
	switch s.Unit {
	case Byte:
		return int64(s.Value)
	case KiB:
		return int64(s.Value * bytesInKiB)
	case MiB:
		return int64(s.Value * bytesInMiB)
	case GiB:
		return int64(s.Value * bytesInGiB)
	case TiB:
		return int64(s.Value * bytesInTiB)
	case PiB:
		return int64(s.Value * bytesInPiB)
	default:
		return 0
	}
}

// Convert converts the size to the target unit
func (s Size) Convert(target Unit) Size {
	bytes := s.ToBytes()

	switch target {
	case Byte:
		return Size{Value: float64(bytes), Unit: Byte}
	case KiB:
		return Size{Value: float64(bytes) / bytesInKiB, Unit: KiB}
	case MiB:
		return Size{Value: float64(bytes) / bytesInMiB, Unit: MiB}
	case GiB:
		return Size{Value: float64(bytes) / bytesInGiB, Unit: GiB}
	case TiB:
		return Size{Value: float64(bytes) / bytesInTiB, Unit: TiB}
	case PiB:
		return Size{Value: float64(bytes) / bytesInPiB, Unit: PiB}
	default:
		return Size{Value: float64(bytes), Unit: Byte}
	}
}

// FromBytes creates a Size from bytes and converts to the best fitting unit
func FromBytes(bytes int64) Size {
	if bytes < bytesInKiB {
		return Size{Value: float64(bytes), Unit: Byte}
	} else if bytes < bytesInMiB {
		return Size{Value: float64(bytes) / bytesInKiB, Unit: KiB}
	} else if bytes < bytesInGiB {
		return Size{Value: float64(bytes) / bytesInMiB, Unit: MiB}
	} else if bytes < bytesInTiB {
		return Size{Value: float64(bytes) / bytesInGiB, Unit: GiB}
	} else if bytes < bytesInPiB {
		return Size{Value: float64(bytes) / bytesInTiB, Unit: TiB}
	} else {
		return Size{Value: float64(bytes) / bytesInPiB, Unit: PiB}
	}
}

// FromString parses a size string like "5MiB" or "2.5 GiB"
func FromString(s string) (Size, error) {
	s = strings.TrimSpace(s)

	// Find the split between number and unit
	var i int
	for i = 0; i < len(s); i++ {
		if (s[i] < '0' || s[i] > '9') && s[i] != '.' {
			break
		}
	}

	if i == 0 || i == len(s) {
		return Size{}, fmt.Errorf("invalid size format: %s", s)
	}

	valueStr := strings.TrimSpace(s[:i])
	unitStr := strings.TrimSpace(s[i:])

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return Size{}, fmt.Errorf("invalid size value: %s", valueStr)
	}

	unitStr = strings.ToUpper(unitStr)
	unit := Unit(unitStr)

	// Validate the unit
	switch unit {
	case Byte, KiB, MiB, GiB, TiB, PiB:
		return Size{Value: value, Unit: unit}, nil
	default:
		return Size{}, fmt.Errorf("unknown unit: %s", unitStr)
	}
}
