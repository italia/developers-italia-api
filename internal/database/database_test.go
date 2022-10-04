package database

import "testing"

func TestWrapErrors(t *testing.T) {
	type args struct {
		dbError error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{},
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WrapErrors(tt.args.dbError); (err != nil) != tt.wantErr {
				t.Errorf("WrapErrors() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
