// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package kafka

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteRuntime(t *testing.T) {

	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	codecOptions := WriteCodecOptions{
		IncludeMsgpack: true,
		IncludeCBOR:    true,
		IncludeYAML:    true,
		IncludeXML:     true,
	}
	if err := WriteCodec(runtimeDir, "kafka", codecOptions); err != nil {
		t.Fatalf("WriteCodec() error = %v", err)
	}
	if err := WriteProduce(runtimeDir, "kafka"); err != nil {
		t.Fatalf("WriteProduce() error = %v", err)
	}
	if err := WriteRecord(runtimeDir, "kafka"); err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}
	if err := WritePoll(runtimeDir, "kafka"); err != nil {
		t.Fatalf("WritePoll() error = %v", err)
	}
	if err := WriteSecurity(runtimeDir, "kafka"); err != nil {
		t.Fatalf("WriteSecurity() error = %v", err)
	}

	wantSymbols := map[string][]string{
		"codec.go":    {"type codec interface", `codecs["msgpack"]`, `codecs["cbor"]`, `codecs["yaml"]`, `codecs["xml"]`},
		"produce.go":  {"func produceAndWait", "func joinOutcomes"},
		"record.go":   {"type Meta struct", "func HeaderValue", "AtStart"},
		"poll.go":     {"func groupRecordsByTopic", "func sortedTopics", "func dispatchTopics", "type TopicHandler"},
		"security.go": {"func saslMechanism", `case "PLAIN"`, `case "SCRAM-SHA-256"`, `case "SCRAM-SHA-512"`},
	}
	for name, symbols := range wantSymbols {
		content, err := os.ReadFile(filepath.Join(runtimeDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(content)
		for _, symbol := range symbols {
			if !strings.Contains(text, symbol) {
				t.Errorf("%s does not contain %q", name, symbol)
			}
		}
		if name == "poll.go" {
			for _, forbidden := range []string{"CommitMode", "commitModeAfterBatch", "commitModeAuto"} {
				if strings.Contains(text, forbidden) {
					t.Errorf("poll.go must not contain unused %q", forbidden)
				}
			}
		}
	}
	runtimeTest := `package kafka

import (
	"errors"
	"reflect"
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
)

func TestGeneratedRuntimeBehavior(t *testing.T) {

	source := []byte("original")
	var destination []byte
	if err := (bytesCodec{}).Unmarshal(source, &destination); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	source[0] = 'X'
	if string(destination) != "original" {
		t.Fatalf("destination changed with source: %q", destination)
	}
	topics := sortedTopics(map[string][]*kgo.Record{
		"zebra": {{Topic: "zebra"}},
		"alpha": {{Topic: "alpha"}},
		"empty": nil,
	})
	if want := []string{"alpha", "zebra"}; !reflect.DeepEqual(topics, want) {
		t.Fatalf("sortedTopics() = %v, want %v", topics, want)
	}
	first := errors.New("first")
	second := errors.New("second")
	joined := joinOutcomes([]produceOutcome{{err: first}, {}, {err: second}})
	if !errors.Is(joined, first) || !errors.Is(joined, second) {
		t.Fatalf("joinOutcomes() = %v, want both errors", joined)
	}
}
`
	if err := os.WriteFile(filepath.Join(runtimeDir, "runtime_test.go"), []byte(runtimeTest), 0o644); err != nil {
		t.Fatalf("write runtime test: %v", err)
	}

	goMod := "module example.test/generated\n\ngo 1.26\n\nrequire github.com/twmb/franz-go v1.21.5\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go test ./...: %v\n%s", err, output)
	}
}
