package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDSNToURL(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		want    string
		wantErr string
	}{
		{
			name:    "empty DSN",
			dsn:     "",
			wantErr: "DATABASE_DSN is not set",
		},
		{
			name: "postgres:// URL passes through",
			dsn:  "postgres://user:pass@host:5432/mydb?sslmode=require",
			want: "postgres://user:pass@host:5432/mydb?sslmode=require",
		},
		{
			name: "postgresql:// URL passes through",
			dsn:  "postgresql://user:pass@host:5432/mydb",
			want: "postgresql://user:pass@host:5432/mydb",
		},
		{
			name: "key=value DSN",
			dsn:  "host=myhost port=5432 dbname=mydb user=myuser password=mypassword sslmode=require",
			want: "postgres://myuser:mypassword@myhost:5432/mydb?sslmode=require",
		},
		{
			name: "key=value DSN without port defaults to 5432",
			dsn:  "host=myhost dbname=mydb user=myuser password=mypassword",
			want: "postgres://myuser:mypassword@myhost:5432/mydb",
		},
		{
			name: "key=value DSN without sslmode omits query string",
			dsn:  "host=myhost port=5432 dbname=mydb user=myuser password=mypassword",
			want: "postgres://myuser:mypassword@myhost:5432/mydb",
		},
		{
			name: "key=value DSN with special chars in password",
			dsn:  "host=myhost port=5432 dbname=mydb user=myuser password=p@ss:w0rd!",
			want: "postgres://myuser:p%40ss%3Aw0rd%21@myhost:5432/mydb",
		},
		{
			name: "key=value DSN with IPv6 host",
			dsn:  "host=::1 port=5432 dbname=mydb user=myuser password=mypassword",
			want: "postgres://myuser:mypassword@[::1]:5432/mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DSNToURL(tt.dsn)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
