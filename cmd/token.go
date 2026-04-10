package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/italia/developers-italia-api/internal/common"
	"github.com/o1egl/paseto"
	"github.com/spf13/cobra"
)

var errInvalidKeyLength = errors.New("invalid key length")

func NewTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage PASETO tokens",
	}

	cmd.AddCommand(newTokenCreateCmd())

	return cmd
}

func newTokenCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "create",
		Short:        "Create a PASETO v2 token",
		SilenceUsage: true,
		RunE:         runTokenCreate,
	}

	cmd.Flags().String("key", "", "base64-encoded 32-byte symmetric key (generates one if empty)")
	cmd.Flags().Duration("expiry", 365*24*time.Hour, "token expiry duration (0 = never expires)")

	cmd.Flags().Lookup("expiry").DefValue = "1 year"

	cmd.Flags().String("sub", "", "token subject, identifies the caller (optional)")

	return cmd
}

func runTokenCreate(cmd *cobra.Command, _ []string) error {
	keyStr, _ := cmd.Flags().GetString("key")
	expiry, _ := cmd.Flags().GetDuration("expiry")
	subject, _ := cmd.Flags().GetString("sub")

	var key []byte

	if keyStr == "" {
		key = make([]byte, common.SymmetricKeyLen)

		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("can't generate key: %w", err)
		}

		encoded := base64.StdEncoding.EncodeToString(key)

		fmt.Fprintf(os.Stderr, "No --key passed, generating random PASETO secret key: %s\n\n", encoded)
	} else {
		var err error

		key, err = base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			return fmt.Errorf("can't decode key: %w", err)
		}

		if len(key) != common.SymmetricKeyLen {
			return fmt.Errorf("%w: must be %d bytes, got %d", errInvalidKeyLength, common.SymmetricKeyLen, len(key))
		}
	}

	now := time.Now().UTC()
	payload := paseto.JSONToken{
		IssuedAt: now,
		Subject:  subject,
	}

	if expiry > 0 {
		payload.Expiration = now.Add(expiry)
	}

	token, err := paseto.NewV2().Encrypt(key, payload, nil)
	if err != nil {
		return fmt.Errorf("can't create token: %w", err)
	}

	payloadJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("can't marshal token payload: %w", err)
	}

	fmt.Fprintf(os.Stderr, "claims:\n%s\n\n", payloadJSON)
	fmt.Fprintf(os.Stderr, "token:\n")

	os.Stdout.WriteString(token + "\n")

	fmt.Fprint(os.Stderr, "\npermissions:\n"+
		"   - software:    create, update, delete\n"+
		"   - publishers:  create, update, delete\n"+
		"   - logs:        create, update, delete\n"+
		"   - webhooks:    create, update, delete")

	return nil
}
