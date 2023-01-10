package gslog

import (
	"bytes"
	"cloud.google.com/go/logging"
	"context"
	"errors"
	"golang.org/x/exp/slog"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGoogleHandler_Handle(t *testing.T) {
	t.Cleanup(func() {

	})
	client, err := logging.NewClient(context.Background(), os.Getenv("GCP_PROJECT"))
	if err != nil {
		t.Fatal(err)
	}

	str := `some 
thing bad\n "happened"`

	var b []byte

	type fields struct {
		conf   GCPConfig
		logger *logging.Logger
		opts   *slog.HandlerOptions
	}
	type args struct {
		err     error
		attrs   []any
		logType slog.Level
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "with group and attr logging",
			fields: fields{
				conf: GCPConfig{
					logger: client.Logger("gcp-log-test"),
					w:      bytes.NewBuffer(b),
				},
			},
			args: args{
				err:     errors.New(str),
				logType: slog.LevelError,
			},
			want:    `","level":"ERROR","msg":"oops","my_app":{"key1":"value1","key2":"value2","err":"some \nthing bad\\n \"happened\""}}`,
			wantErr: false,
		},
		{
			name: "disabled",
			fields: fields{
				conf: GCPConfig{
					w:      bytes.NewBuffer(b),
					logger: client.Logger("gcp-log-test"),
				},
			},
			args: args{
				attrs: []any{
					"title", "some title",
					"height", 100,
					"with", 99.9,
				},
				logType: slog.LevelDebug,
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "debug logging",
			fields: fields{
				conf: GCPConfig{
					w:      bytes.NewBuffer(b),
					logger: client.Logger("gcp-log-test"),
					opts:   &slog.HandlerOptions{Level: slog.LevelDebug},
				},
			},
			args: args{
				attrs: []any{
					"title", "some title",
					"height", 100,
					"with", 99.9,
				},
				logType: slog.LevelDebug,
			},
			want:    `","level":"DEBUG","msg":"oops","my_app":{"key1":"value1","key2":"value2","title":"some title","height":100,"with":99.9}}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := slog.New(NewGCPHandler(tt.fields.conf))

			switch tt.args.logType {
			case slog.LevelError:
				log.WithGroup("my_app").With(
					"key1", "value1",
					"key2", "value2",
				).Error("oops", tt.args.err)
			case slog.LevelDebug:
				log.WithGroup("my_app").With(
					"key1", "value1",
					"key2", "value2",
				).Debug("oops", tt.args.attrs...)
			}

			var res []byte

			if tt.want != "" {
				res, err = io.ReadAll(tt.fields.conf.w)
				res = res[41:]
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Error() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := strings.Trim(string(res), "\n")
			if got != tt.want {
				t.Errorf("Error() got = %s, want %s", string(res), tt.want)
			}
		})
	}
}

//","level":"ERROR","msg":"oops","err":"something bad happened"}
//","level":"ERROR","msg":"oops","err":"something bad happened"}
