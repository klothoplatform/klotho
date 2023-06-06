package logging

import (
	"encoding/base64"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// bufferEncoder wraps a buffer conforming to the various encoder interfaces
type bufferEncoder struct {
	b              *buffer.Buffer
	openNamespaces int
}

func (w *bufferEncoder) Append(i interface{}) {
	w.appendSep()
	fmt.Fprint(w.b, i)
}
func (w *bufferEncoder) appendSep() {
	last := w.b.Len() - 1
	if last < 0 {
		return
	}
	switch w.b.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		w.b.AppendString(", ")
	}
}

// PrimitiveArrayEncoder
func (w *bufferEncoder) AppendBool(b bool)             { w.appendSep(); w.b.AppendBool(b) }
func (w *bufferEncoder) AppendByteString(b []byte)     { w.appendSep(); w.b.AppendString(string(b)) }
func (w *bufferEncoder) AppendComplex128(c complex128) { w.Append(c) }
func (w *bufferEncoder) AppendComplex64(c complex64)   { w.Append(c) }

func (w *bufferEncoder) appendFloat(val float64, bitSize int) {
	w.appendSep()
	switch {
	case math.IsNaN(val):
		w.b.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		w.b.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		w.b.AppendString(`"-Inf"`)
	default:
		w.b.AppendFloat(val, bitSize)
	}
}
func (w *bufferEncoder) AppendFloat64(f float64)   { w.appendFloat(f, 64) }
func (w *bufferEncoder) AppendFloat32(f float32)   { w.appendFloat(float64(f), 32) }
func (w *bufferEncoder) AppendInt(i int)           { w.appendSep(); w.b.AppendInt(int64(i)) }
func (w *bufferEncoder) AppendInt64(i int64)       { w.appendSep(); w.b.AppendInt(i) }
func (w *bufferEncoder) AppendInt32(i int32)       { w.appendSep(); w.b.AppendInt(int64(i)) }
func (w *bufferEncoder) AppendInt16(i int16)       { w.appendSep(); w.b.AppendInt(int64(i)) }
func (w *bufferEncoder) AppendInt8(i int8)         { w.appendSep(); w.b.AppendInt(int64(i)) }
func (w *bufferEncoder) AppendString(s string)     { w.appendSep(); w.b.AppendString(s) }
func (w *bufferEncoder) AppendUint(i uint)         { w.appendSep(); w.b.AppendUint(uint64(i)) }
func (w *bufferEncoder) AppendUint64(i uint64)     { w.appendSep(); w.b.AppendUint(i) }
func (w *bufferEncoder) AppendUint32(i uint32)     { w.appendSep(); w.b.AppendUint(uint64(i)) }
func (w *bufferEncoder) AppendUint16(i uint16)     { w.appendSep(); w.b.AppendUint(uint64(i)) }
func (w *bufferEncoder) AppendUint8(i uint8)       { w.appendSep(); w.b.AppendByte(byte(i)) }
func (w *bufferEncoder) AppendUintptr(ptr uintptr) { w.appendSep(); w.b.AppendUint(uint64(ptr)) }

// ArrayEncoder
func (w *bufferEncoder) AppendDuration(d time.Duration) { w.appendSep(); w.b.AppendString(d.String()) }
func (w *bufferEncoder) AppendTime(t time.Time)         { w.appendSep(); w.b.AppendTime(t, time.RFC3339) }
func (w *bufferEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	w.appendSep()
	w.b.AppendByte('[')
	err := arr.MarshalLogArray(w)
	w.b.AppendByte(']')
	return err
}
func (w *bufferEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	w.appendSep()
	w.b.AppendByte('{')
	err := obj.MarshalLogObject(w)
	w.b.AppendByte('}')
	return err
}
func (w *bufferEncoder) AppendReflected(value interface{}) error {
	w.Append(value)
	return nil
}

func (w *bufferEncoder) addKey(key string) {
	w.appendSep()
	w.b.AppendString(key)
	w.b.AppendString(": ")
}

// ObjectEncoder

func (w *bufferEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	w.addKey(key)
	return w.AppendArray(arr)
}

func (w *bufferEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	w.addKey(key)
	return w.AppendObject(obj)
}

func (w *bufferEncoder) AddBinary(key string, val []byte) {
	w.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (w *bufferEncoder) AddByteString(key string, val []byte) {
	w.addKey(key)
	w.AppendByteString(val)
}

func (w *bufferEncoder) AddBool(key string, val bool) {
	w.addKey(key)
	w.AppendBool(val)
}

func (w *bufferEncoder) AddComplex128(key string, val complex128) {
	w.addKey(key)
	w.AppendComplex128(val)
}

func (w *bufferEncoder) AddDuration(key string, val time.Duration) {
	w.addKey(key)
	w.AppendDuration(val)
}

func (w *bufferEncoder) AddFloat64(key string, val float64) {
	w.addKey(key)
	w.AppendFloat64(val)
}

func (w *bufferEncoder) AddFloat32(key string, val float32) {
	w.addKey(key)
	w.AppendFloat32(val)
}

func (w *bufferEncoder) AddInt64(key string, val int64) {
	w.addKey(key)
	w.AppendInt64(val)
}

func (w *bufferEncoder) AddReflected(key string, obj interface{}) error {
	w.addKey(key)
	return w.AppendReflected(obj)
}

func (w *bufferEncoder) OpenNamespace(key string) {
	w.addKey(key)
	w.b.AppendByte('{')
	w.openNamespaces++
}

func (w *bufferEncoder) AddString(key, val string) {
	w.addKey(key)
	w.AppendString(val)
}

func (w *bufferEncoder) AddTime(key string, val time.Time) {
	w.addKey(key)
	w.AppendTime(val)
}

func (w *bufferEncoder) AddUint64(key string, val uint64) {
	w.addKey(key)
	w.AppendUint64(val)
}

func (w *bufferEncoder) AddComplex64(k string, v complex64) { w.AddComplex128(k, complex128(v)) }
func (w *bufferEncoder) AddInt(k string, v int)             { w.AddInt64(k, int64(v)) }
func (w *bufferEncoder) AddInt32(k string, v int32)         { w.AddInt64(k, int64(v)) }
func (w *bufferEncoder) AddInt16(k string, v int16)         { w.AddInt64(k, int64(v)) }
func (w *bufferEncoder) AddInt8(k string, v int8)           { w.AddInt64(k, int64(v)) }
func (w *bufferEncoder) AddUint(k string, v uint)           { w.AddUint64(k, uint64(v)) }
func (w *bufferEncoder) AddUint32(k string, v uint32)       { w.AddUint64(k, uint64(v)) }
func (w *bufferEncoder) AddUint16(k string, v uint16)       { w.AddUint64(k, uint64(v)) }
func (w *bufferEncoder) AddUint8(k string, v uint8)         { w.AddUint64(k, uint64(v)) }
func (w *bufferEncoder) AddUintptr(k string, v uintptr)     { w.AddUint64(k, uint64(v)) }
