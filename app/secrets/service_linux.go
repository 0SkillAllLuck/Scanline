//go:build linux

package secrets

import (
	"strings"
	"sync"

	"github.com/0skillallluck/scanline/internal/gettext"
	golibsecret "github.com/lescuer97/go-libsecret"
)

var (
	schemaOnce sync.Once
	schema     *golibsecret.Schema
	schemaErr  error
)

func getSchema() (*golibsecret.Schema, error) {
	schemaOnce.Do(func() {
		schema, schemaErr = golibsecret.NewSchema("dev.skillless.Scanline", golibsecret.SchemaFlagsNone, map[string]golibsecret.SchemaAttributeType{
			"key": golibsecret.SchemaAttributeString,
		})
	})
	return schema, schemaErr
}

type serviceLinux struct{}

func (s *serviceLinux) Available() *ServiceError {
	schema, err := getSchema()
	if err != nil {
		return &ServiceError{
			Title: gettext.Get("Secret Service Error"),
			Body:  gettext.Get("An unknown error occurred when checking for a secret service provider.\n\nSigning in may or may not work. Please see the raw error message for more details:\n\n%s", err.Error()),
			Fatal: false,
		}
	}

	// Fake secret fetch to see if the service is available
	attrs := golibsecret.NewAttributes()
	attrs.Set("key", "dummy_key") //nolint:errcheck
	_, err = golibsecret.PasswordLookupSync(schema, attrs)
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "name is not activatable") || strings.Contains(err.Error(), "ServiceUnknown") {
		return &ServiceError{
			Title: gettext.Get("Secret Service Unavailable"),
			Body:  gettext.Get("No secret service provider is available.\n\nScanline will not be able to store any authentication-related data and you will not be able to sign in.\n\nPlease install a secret service provider such as GNOME Keyring or KDE Wallet."),
			Fatal: false,
		}
	}

	if strings.Contains(err.Error(), "user interaction failed") {
		return &ServiceError{
			Title: gettext.Get("Secret Service Issue"),
			Body:  gettext.Get("Your secret service provider was found, but refused to interact with Scanline.\n\nThis could because you cancelled a prompt or, if you are using a Flatpak, the provider not implementing the XDG Secret Portal service.\n\nScanline will not be able to store any authentication-related data and you will not be able to sign in."),
			Fatal: false,
		}
	}

	return &ServiceError{
		Title: gettext.Get("Secret Service Error"),
		Body:  gettext.Get("An unknown error occurred when checking for a secret service provider.\n\nSigning in may or may not work. Please see the raw error message for more details:\n\n%s", err.Error()),
		Fatal: false,
	}
}

func (s *serviceLinux) Delete(key string) error {
	schema, err := getSchema()
	if err != nil {
		return err
	}
	_, err = golibsecret.ClearPassword(schema, map[string]string{
		"key": key,
	})
	return err
}

func (s *serviceLinux) Get(key string) (Item, error) {
	schema, err := getSchema()
	if err != nil {
		return Item{}, err
	}
	attrs := golibsecret.NewAttributes()
	attrs.Set("key", key) //nolint:errcheck
	val, err := golibsecret.PasswordLookupSync(schema, attrs)
	if val == "" && err == nil {
		return Item{}, ErrKeyNotFound
	}
	return Item{Label: "", Password: val}, err
}

func (s *serviceLinux) Has(key string) (bool, error) {
	_, err := s.Get(key)
	if err == ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *serviceLinux) Set(key string, value Item) error {
	schema, err := getSchema()
	if err != nil {
		return err
	}
	attrs := golibsecret.NewAttributes()
	attrs.Set("key", key) //nolint:errcheck
	return golibsecret.PasswordStoreSync(schema, attrs, golibsecret.CollectionDefault, value.Label, value.Password)
}

func newService() Service {
	return &serviceLinux{}
}
