// Package yaml implements YAML support for the Go language.
//
// It contains the source code for the `github.com/go-yaml/yaml` package.
//
// This is a copy of the original package (v3.0.1) with some code
// (Marshaling/Encoding) removed (reported by the `deadcode` tool
// https://go.dev/blog/deadcode and golangci-lint 'unused' linter).
//
// Before:
//
//	Language                     files          blank        comment           code
//	-------------------------------------------------------------------------------
//	Go                              19           1345           2775          12769
//	Markdown                         1             39              0            111
//	YAML                             1              4              0             57
//	-------------------------------------------------------------------------------
//	SUM:                            21           1388           2775          12937
//
// After (approximate):
//
//	Language                     files          blank        comment           code
//	-------------------------------------------------------------------------------
//	Go                              11           1309           1019           4838
//	-------------------------------------------------------------------------------
//	SUM:                            11           1309           1019           4838
//
// It has been placed here to eliminate the need to download the package from
// remote galaxies.
// Updating this package is not necessary, as it is highly stable and mature.
//
// -------------------------------------------------------------------------------
//
// Copyright (c) 2011-2019 Canonical Ltd
// Copyright (c) 2006-2010 Kirill Simonov
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package yaml
