//go:build darwin

package secrets

import (
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/keybase/go-keychain"
)

const (
	serviceName = "dev.dergs.Tonearm"
)

type serviceDarwin struct{}

func (s *serviceDarwin) Available() *ServiceError {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(serviceName)
	item.SetAccount("dummy_key")

	_, err := keychain.QueryItem(item)
	if err != nil && err != keychain.ErrorItemNotFound {
		return &ServiceError{
			Title: gettext.Get("Keychain Unavailable"),
			Body:  gettext.Get("The macOS Keychain is not available or accessible.\n\nTonearm will not be able to store any authentication-related data and you will not be able to sign in.\n\nPlease ensure your keychain is unlocked and accessible."),
			Fatal: false,
		}
	}
	return nil
}

func (s *serviceDarwin) Delete(key string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(serviceName)
	item.SetAccount(key)

	if err := keychain.DeleteItem(item); err != keychain.ErrorItemNotFound {
		return err
	}

	return ErrKeyNotFound
}

func (s *serviceDarwin) Get(key string) (Item, error) {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(serviceName)
	item.SetAccount(key)
	item.SetMatchLimit(keychain.MatchLimitOne)
	item.SetReturnData(true)

	items, err := keychain.QueryItem(item)
	if err != nil {
		if err == keychain.ErrorItemNotFound {
			return Item{}, ErrKeyNotFound
		}
		return Item{}, err
	}
	if len(items) == 0 {
		return Item{}, ErrKeyNotFound
	}

	return Item{
		Label:    items[0].Label,
		Password: string(items[0].Data),
	}, nil
}

func (s *serviceDarwin) Has(key string) (bool, error) {
	_, err := s.Get(key)
	if err == ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *serviceDarwin) Set(key string, value Item) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(serviceName)
	item.SetAccount(key)
	item.SetLabel(value.Label)
	item.SetData([]byte(value.Password))
	item.SetSynchronizable(keychain.AccessibleWhenUnlocked)

	if err := keychain.AddItem(item); err == keychain.ErrorDuplicateItem {
		query := keychain.NewItem()
		query.SetSecClass(keychain.SecClassGenericPassword)
		query.SetService(serviceName)
		query.SetAccount(key)
		return keychain.UpdateItem(query, item)
	} else {
		return err
	}
}

func newService() Service {
	return &serviceDarwin{}
}
