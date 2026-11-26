// Copyright 2024 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/gohugoio/hugo/common/hexec"
	"github.com/gohugoio/hugo/common/loggers"
	"github.com/gohugoio/hugo/config/security"
)

func TestIsUnquotedCSSValue(t *testing.T) {
	c := qt.New(t)

	for _, test := range []struct {
		in  any
		out bool
	}{
		{"24px", true},
		{"1.5rem", true},
		{"10%", true},
		{"hsl(0, 0%, 100%)", true},
		{"calc(24px + 36px)", true},
		{"24xxx", true}, // a false positive.
		{123, true},
		{123.12, true},
		{"#fff", true},
		{"#ffffff", true},
		{"#ffffffff", false},
	} {
		c.Assert(isTypedCSSValue(test.in), qt.Equals, test.out)
	}
}

func TestFindFile(t *testing.T) {
	c := qt.New(t)

	base := t.TempDir()
	site := filepath.Join(base, "site")
	assets := filepath.Join(site, "assets")
	c.Assert(os.MkdirAll(filepath.Join(assets, "dir"), 0o755), qt.IsNil)
	c.Assert(os.WriteFile(filepath.Join(assets, "_a.scss"), nil, 0o644), qt.IsNil)
	secret := filepath.Join(base, "secret.scss")
	c.Assert(os.WriteFile(secret, nil, 0o644), qt.IsNil)

	ex := hexec.New(security.DefaultConfig, site, loggers.NewDefault())
	const outsideErr = `"[^"]*secret.scss" resolves to "[^"]*secret.scss" outside the project.*allowRead.*`

	filename, err := FindFile(ex, []string{assets}, []string{"a.scss", "_a.scss"})
	c.Assert(err, qt.IsNil)
	c.Assert(filename, qt.Equals, filepath.Join(assets, "_a.scss"))

	filename, err = FindFile(ex, []string{assets}, []string{"b.scss", "dir"})
	c.Assert(err, qt.IsNil)
	c.Assert(filename, qt.Equals, "")

	_, err = FindFile(ex, []string{assets}, []string{filepath.Join("..", "..", "secret.scss")})
	c.Assert(err, qt.ErrorMatches, outsideErr)

	_, err = FindFile(ex, []string{assets}, []string{secret})
	c.Assert(err, qt.ErrorMatches, outsideErr)

	// Rooted, but without a volume name on Windows.
	wd, err := os.Getwd()
	c.Assert(err, qt.IsNil)
	c.Assert(os.Chdir(base), qt.IsNil)
	t.Cleanup(func() { os.Chdir(wd) })
	_, err = FindFile(ex, []string{assets}, []string{strings.TrimPrefix(secret, filepath.VolumeName(secret))})
	c.Assert(err, qt.ErrorMatches, outsideErr)
}
