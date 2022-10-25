/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package applier

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"reflect"
	"testing"
)

// collector is a commandSite implementation that stubs cmd.Run() calls for tests
type collector struct {
	Error error
	Cmds  []*exec.Cmd
}

func (s *collector) Run(c *exec.Cmd) error {
	s.Cmds = append(s.Cmds, c)
	return s.Error
}

func TestKubectlApply(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		manifest   string
		validate   bool
		args       []string
		err        error
		expectArgs []string
	}{
		{
			name:       "manifest",
			namespace:  "",
			manifest:   "foo",
			expectArgs: []string{"kubectl", "apply", "--validate=false", "-f", "-"},
		},
		{
			name:       "manifest with apply",
			namespace:  "kube-system",
			manifest:   "heynow",
			expectArgs: []string{"kubectl", "apply", "-n", "kube-system", "--validate=false", "-f", "-"},
		},
		{
			name:       "manifest with validate",
			namespace:  "",
			manifest:   "foo",
			validate:   true,
			expectArgs: []string{"kubectl", "apply", "--validate=true", "-f", "-"},
		},
		{
			name:       "error propagation",
			expectArgs: []string{"kubectl", "apply", "--validate=false", "-f", "-"},
			err:        errors.New("error"),
		},
		{
			name:       "manifest with prune",
			namespace:  "kube-system",
			manifest:   "heynow",
			args:       []string{"--prune=true", "--prune-whitelist=hello-world"},
			expectArgs: []string{"kubectl", "apply", "-n", "kube-system", "--validate=false", "--prune=true", "--prune-whitelist=hello-world", "-f", "-"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cs := collector{Error: test.err}
			kubectl := &ExecKubectl{cmdSite: &cs}
			opts := ApplierOptions{
				Namespace: test.namespace,
				Manifest:  test.manifest,
				Validate:  test.validate,
				ExtraArgs: test.args,
			}
			err := kubectl.Apply(context.Background(), opts)

			if test.err != nil && err == nil {
				t.Error("expected error to occur")
			} else if test.err == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(cs.Cmds) != 1 {
				t.Errorf("expected 1 command to be invoked, got: %d", len(cs.Cmds))
			}

			cmd := cs.Cmds[0]
			if !reflect.DeepEqual(cmd.Args, test.expectArgs) {
				t.Errorf("argument mistmatch, expected: %v, got: %v", test.expectArgs, cmd.Args)
			}

			stdinBytes, err := io.ReadAll(cmd.Stdin)
			if err != nil {
				t.Fatalf("error reading manifest from stdin: %v", err)
			}
			if stdin := string(stdinBytes); stdin != test.manifest {
				t.Errorf("manifest mismatch, expected: %v, got: %v", test.manifest, stdin)
			}
		})
	}
}

func TestKubectlApplier(t *testing.T) {
	applier := NewExec()
	runApplierGoldenTests(t, "testdata/kubectl", true, applier)
}
