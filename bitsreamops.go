/**
 * Go Bitstream Ops
 *
 *    Copyright 2017 Tenta, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * For any questions, please contact developer@tenta.io
 *
 * bitstreamops.go: Bitstream wrapper and operations
 */

// Provides some basic wrapper interfaces around byte arrays to operate on them as bitstreams.
package bitstreamops

import (
	"errors"
	"fmt"
	"io/ioutil"
)

type BitStreamOps struct {
	buf             []byte
	index, bitindex uint
}

func NewBitStreamOps() *BitStreamOps {
	return &BitStreamOps{make([]byte, 1), 0, 0}
}

func NewBitStreamOpsReader(b []byte) *BitStreamOps {
	return &BitStreamOps{b, 0, 0}
}

func (b *BitStreamOps) Buffer() []byte {
	remaind := 0
	if b.bitindex == 0 {
		remaind = 1
	}
	return b.buf[:len(b.buf)-remaind]
}

func (b *BitStreamOps) Index() uint {
	return b.index
}

func (b *BitStreamOps) BIndex() uint {
	return b.bitindex
}

/*
** Does not check end of buffer (so that it can be used for r/w)
 */
func (b *BitStreamOps) JumpToNextByte() {
	if b.bitindex == 0 {
		return
	}
	b.bitindex = 0
	b.index++
	b.buf = append(b.buf, 0)
	return

}

func (b *BitStreamOps) JumpToNextByteForRead() {
	if b.bitindex == 0 {
		return
	}
	b.bitindex = 0
	b.index++
	return
}

func (b *BitStreamOps) CollectAll() []byte {
	b.JumpToNextByteForRead()
	return b.buf[b.index:]
}

func (b *BitStreamOps) HasMoreBytes() bool {
	return int(b.index) < len(b.buf)-1
}

/// Append and Concat assumes byte boundaries are all settled and in order
func (b *BitStreamOps) Append(s []byte) {
	b.buf = append(b.Buffer(), s...)
	b.index += uint(len(s))
	b.buf = append(b.buf, 0)
}

func (b *BitStreamOps) Concat(s string) {
	b.Append([]byte(s))
}

func (b *BitStreamOps) Emit(val uint, numbits int) (err error) {
	if numbits < 1 || numbits > 32 {
		return errors.New("Invalid parameter value")
	}
	for i := numbits - 1; i >= 0; i-- {
		if b.bitindex == 8 {
			b.buf = append(b.buf, 0)
			b.index++
			b.bitindex = 0
		}

		b.buf[b.index] |= byte(((val & (1 << uint(i))) >> uint(i)) << (7 - b.bitindex))
		b.bitindex++

		if b.bitindex == 8 {
			b.buf = append(b.buf, 0)
			b.index++
			b.bitindex = 0
		}
	}

	return nil
}

/*
** Call JumpToNextByte before calling emit on data more than 1 bit
 */
func (b *BitStreamOps) EmitByte(val uint8) {
	b.buf[b.index] = val
	b.buf = append(b.buf, 0)
	b.index++
	b.bitindex = 0
}

func (b *BitStreamOps) EmitWord(val uint16) {
	b.EmitByte(uint8((val & 0xff00) >> 8))
	b.EmitByte(uint8(val & 0x00ff))
}

func (b *BitStreamOps) EmitDWord(val uint32) {
	b.EmitWord(uint16((val & 0xffff0000) >> 16))
	b.EmitWord(uint16(val & 0x0000ffff))
}

func (b *BitStreamOps) DeConcat(n int) (string, error) {
	ret := b.buf[int(b.index) : int(b.index)+n]
	b.index += uint(n)
	return string(ret), nil
}

func (b *BitStreamOps) DeAppend(n int) ([]byte, error) {
	ret := b.buf[int(b.index) : int(b.index)+n]
	b.index += uint(n)
	return ret, nil
}

func (b *BitStreamOps) Collect(numbits int) (ret uint, err error) {
	if numbits < 1 || numbits > 32 {
		return 0, errors.New("Invalid parameter value")
	}

	for i := numbits - 1; i >= 0; i-- {
		if b.bitindex == 8 {
			b.index++
			b.bitindex = 0
			if b.bitindex == uint(len(b.buf)) {
				return 0, errors.New("Buffer overrun")
			}
		}
		a := uint((b.buf[b.index]&(1<<(7-b.bitindex)))>>(7-b.bitindex)) << uint(i)
		ret |= a
		b.bitindex++

		if b.bitindex == 8 {
			b.index++
			b.bitindex = 0
			if b.bitindex == uint(len(b.buf)) {
				return 0, errors.New("Buffer overrun")
			}
		}
	}

	return
}

/*
** Jump to next byte boundary before calling Collect on data more than 1 bit
 */
func (b *BitStreamOps) CollectByte() (ret uint, err error) {
	if (b.bitindex != 8 && b.bitindex != 0) || b.index == uint(len(b.buf)) {
		return 0, errors.New(fmt.Sprintf("Constellation not favorable -- len %d, index %d, bitindex %d", len(b.buf), b.index, b.bitindex))
	}
	b.index++
	return uint(b.buf[b.index-1]), nil
}

func (b *BitStreamOps) CollectWord() (ret uint, err error) {
	retMSB, e := b.CollectByte()
	retLSB, f := b.CollectByte()
	if e == nil && f == nil {
		return uint(retMSB<<8 | retLSB), nil
	}
	return 0, f
}

func (b *BitStreamOps) CollectDWord() (ret uint, err error) {
	retMSW, e := b.CollectWord()
	retLSW, f := b.CollectWord()
	if e == nil && f == nil {
		return uint(retMSW<<16 | retLSW), nil
	}
	return 0, f
}

func (b *BitStreamOps) WriteToFile(fname string) (err error) {
	return ioutil.WriteFile(fname, b.buf, 0x0777)
}
