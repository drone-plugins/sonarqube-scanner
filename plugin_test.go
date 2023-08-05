package main

import (
	"reflect"
	"testing"
)

func TestParseJunit(t *testing.T) {
	type args struct {
		projectArray Project
		projectName  string
	}
	tests := []struct {
		name string
		args args
		want Testsuites
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseJunit(tt.args.projectArray, tt.args.projectName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseJunit() = %v, want %v", got, tt.want)
			}
		})
	}
}
