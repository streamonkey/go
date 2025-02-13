// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains tests for sizes.

package types2_test

import (
	"cmd/compile/internal/syntax"
	"cmd/compile/internal/types2"
	"testing"
)

// findStructType typechecks src and returns the first struct type encountered.
func findStructType(t *testing.T, src string) *types2.Struct {
	return findStructTypeConfig(t, src, &types2.Config{})
}

func findStructTypeConfig(t *testing.T, src string, conf *types2.Config) *types2.Struct {
	types := make(map[syntax.Expr]types2.TypeAndValue)
	mustTypecheck("x", src, &types2.Info{Types: types})
	for _, tv := range types {
		if ts, ok := tv.Type.(*types2.Struct); ok {
			return ts
		}
	}
	t.Fatalf("failed to find a struct type in src:\n%s\n", src)
	return nil
}

// Issue 16316
func TestMultipleSizeUse(t *testing.T) {
	const src = `
package main

type S struct {
    i int
    b bool
    s string
    n int
}
`
	ts := findStructType(t, src)
	sizes := types2.StdSizes{WordSize: 4, MaxAlign: 4}
	if got := sizes.Sizeof(ts); got != 20 {
		t.Errorf("Sizeof(%v) with WordSize 4 = %d want 20", ts, got)
	}
	sizes = types2.StdSizes{WordSize: 8, MaxAlign: 8}
	if got := sizes.Sizeof(ts); got != 40 {
		t.Errorf("Sizeof(%v) with WordSize 8 = %d want 40", ts, got)
	}
}

// Issue 16464
func TestAlignofNaclSlice(t *testing.T) {
	const src = `
package main

var s struct {
	x *int
	y []byte
}
`
	ts := findStructType(t, src)
	sizes := &types2.StdSizes{WordSize: 4, MaxAlign: 8}
	var fields []*types2.Var
	// Make a copy manually :(
	for i := 0; i < ts.NumFields(); i++ {
		fields = append(fields, ts.Field(i))
	}
	offsets := sizes.Offsetsof(fields)
	if offsets[0] != 0 || offsets[1] != 4 {
		t.Errorf("OffsetsOf(%v) = %v want %v", ts, offsets, []int{0, 4})
	}
}

func TestIssue16902(t *testing.T) {
	const src = `
package a

import "unsafe"

const _ = unsafe.Offsetof(struct{ x int64 }{}.x)
`
	f := mustParse("x.go", src)
	info := types2.Info{Types: make(map[syntax.Expr]types2.TypeAndValue)}
	conf := types2.Config{
		Importer: defaultImporter(),
		Sizes:    &types2.StdSizes{WordSize: 8, MaxAlign: 8},
	}
	_, err := conf.Check("x", []*syntax.File{f}, &info)
	if err != nil {
		t.Fatal(err)
	}
	for _, tv := range info.Types {
		_ = conf.Sizes.Sizeof(tv.Type)
		_ = conf.Sizes.Alignof(tv.Type)
	}
}

// Issue #53884.
func TestAtomicAlign(t *testing.T) {
	const src = `
package main

import "sync/atomic"

var s struct {
	x int32
	y atomic.Int64
	z int64
}
`

	want := []int64{0, 8, 16}
	for _, arch := range []string{"386", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			conf := types2.Config{
				Importer: defaultImporter(),
				Sizes:    types2.SizesFor("gc", arch),
			}
			ts := findStructTypeConfig(t, src, &conf)
			var fields []*types2.Var
			// Make a copy manually :(
			for i := 0; i < ts.NumFields(); i++ {
				fields = append(fields, ts.Field(i))
			}

			offsets := conf.Sizes.Offsetsof(fields)
			if offsets[0] != want[0] || offsets[1] != want[1] || offsets[2] != want[2] {
				t.Errorf("OffsetsOf(%v) = %v want %v", ts, offsets, want)
			}
		})
	}
}
