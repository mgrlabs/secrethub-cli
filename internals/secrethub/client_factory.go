package secrethub

import (
	"net/url"
	"strings"

	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/configdir"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
)

// Errors
var (
	ErrUnknownIdentityProvider = errMain.Code("unknown_identity_provider").ErrorPref("%s is not a supported identity provider. Valid options are `aws` and `key`.")
)

// ClientFactory handles creating a new client with the configured options.
type ClientFactory interface {
	// NewClient returns a new SecretHub client.
	NewClient() (secrethub.ClientInterface, error)
	NewClientWithCredentials(credentials.Provider) (secrethub.ClientInterface, error)
	NewUnauthenticatedClient() (secrethub.ClientInterface, error)
	Register(FlagRegisterer)
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(store CredentialConfig) ClientFactory {
	return &clientFactory{
		store: store,
	}
}

type clientFactory struct {
	client           *secrethub.Client
	ServerURL        *url.URL
	identityProvider string
	store            CredentialConfig
}

// Register the flags for configuration on a cli application.
func (f *clientFactory) Register(r FlagRegisterer) {
	r.Flag("api-remote", "The SecretHub API address, don't set this unless you know what you're doing.").Hidden().URLVar(&f.ServerURL)
	r.Flag("identity-provider", "Enable native authentication with a trusted identity provider. Options are `aws` (IAM + KMS) and `key`. When you run the CLI on one of the platforms, you can leverage their respective identity providers to do native keyless authentication. Defaults to key, which uses the default credential sourced from a file, command-line flag, or environment variable. ").Default("key").StringVar(&f.identityProvider)
}

// NewClient returns a new client that is configured to use the remote that
// is set with the flag.
func (f *clientFactory) NewClient() (secrethub.ClientInterface, error) {
	if f.client == nil {
		var credentialProvider credentials.Provider
		switch strings.ToLower(f.identityProvider) {
		case "aws":
			credentialProvider = credentials.UseAWS()
		case "key":
			credentialProvider = f.store.Provider()
		default:
			return nil, ErrUnknownIdentityProvider(f.identityProvider)
		}

		options := f.baseClientOptions()
		options = append(options, secrethub.WithCredentials(credentialProvider))

		client, err := secrethub.NewClient(options...)
		if err == configdir.ErrCredentialNotFound {
			return nil, ErrCredentialNotExist
		} else if err != nil {
			return nil, err
		}
		f.client = client
	}
	return f.client, nil
}

func (f *clientFactory) NewClientWithCredentials(provider credentials.Provider) (secrethub.ClientInterface, error) {
	options := f.baseClientOptions()
	options = append(options, secrethub.WithCredentials(provider))

	client, err := secrethub.NewClient(options...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (f *clientFactory) NewUnauthenticatedClient() (secrethub.ClientInterface, error) {
	options := f.baseClientOptions()

	client, err := secrethub.NewClient(options...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (f *clientFactory) baseClientOptions() []secrethub.ClientOption {
	options := []secrethub.ClientOption{
		secrethub.WithConfigDir(f.store.ConfigDir()),
		secrethub.WithAppInfo(&secrethub.AppInfo{
			Name:    "secrethub-cli",
			Version: Version,
		}),
	}

	if f.ServerURL != nil {
		options = append(options, secrethub.WithServerURL(f.ServerURL.String()))
	}

	return options
}
