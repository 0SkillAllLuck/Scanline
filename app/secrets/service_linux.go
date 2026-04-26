//go:build linux

package secrets

import (
	"errors"
	"strings"

	"github.com/0skillallluck/scanline/internal/gettext"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"codeberg.org/puregotk/puregotk/v4/secret"
)

var schema *secret.Schema = secret.NewSchema("dev.skillless.Scanline", secret.SchemaNoneValue, "key", secret.SchemaAttributeStringValue)

type serviceLinux struct{}

func (s *serviceLinux) Available() *ServiceError {
	// Fake secret fetch to see if the service is available
	var err *glib.Error
	secret.PasswordLookupSync(schema, nil, &err, "key", "dummy_key")

	if err == nil {
		return nil
	}
	defer err.Free()

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
		Body:  gettext.Getf("An unknown error occurred when checking for a secret service provider.\n\nSigning in may or may not work. Please see the raw error message for more details:\n\n%s", err.Error()),
		Fatal: false,
	}
}

func (s *serviceLinux) Delete(key string) error {
	var err *glib.Error
	secret.PasswordClearSync(schema, nil, &err, "key", key)
	if err != nil {
		defer err.Free()
		return errors.New(err.Error())
	}
	return nil
}

func (s *serviceLinux) Get(key string) (Item, error) {
	var err *glib.Error
	val := secret.PasswordLookupSync(schema, nil, &err, "key", key)

	if err != nil {
		defer err.Free()
		return Item{}, errors.New(err.Error())
	}

	if val == "" {
		return Item{}, ErrKeyNotFound
	}
	return Item{Label: "", Password: val}, nil
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
	var err *glib.Error
	secret.PasswordStoreSync(schema, secret.COLLECTION_DEFAULT, value.Label, value.Password, nil, &err, "key", key)
	if err != nil {
		defer err.Free()
		return errors.New(err.Error())
	}
	return nil
}

func newService() Service {
	return &serviceLinux{}
}
