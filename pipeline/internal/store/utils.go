package store

import (
	"google.golang.org/protobuf/types/known/timestamppb"
)

func safeU64(v uint64) uint64 {
	return v // protobuf getters return 0 if nil
}

func safeF64(v float64) float64 {
	return v // same, safe default
}

func safeTime(ts *timestamppb.Timestamp) int64 {
	if ts == nil {
		return 0
	}
	return ts.AsTime().Unix()
}

func safeStringPtr[T any](ptr *T, getter func(*T) string) string {
	if ptr == nil {
		return ""
	}
	return getter(ptr)
}

func safeInt64Ptr[T any](ptr *T, getter func(*T) int64) int64 {
	if ptr == nil {
		return 0
	}
	return getter(ptr)
}

func safeUInt32Ptr[T any](ptr *T, getter func(*T) uint32) uint32 {
	if ptr == nil {
		return 0
	}
	return getter(ptr)
}

func safeMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
