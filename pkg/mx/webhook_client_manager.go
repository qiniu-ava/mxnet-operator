package mx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"

	lru "github.com/hashicorp/golang-lru"
	"github.com/qiniu-ava/mxnet-operator/pkg/apis/qiniu/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	webhook "k8s.io/apiserver/pkg/admission/plugin/webhook/config"
	webhookerrors "k8s.io/apiserver/pkg/admission/plugin/webhook/errors"
	"k8s.io/client-go/rest"
)

const (
	defaultCacheSize = 200
)

var (
	ErrNeedServiceOrURL = errors.New("webhook configuration must have either service or URL")
)

// ClientManager builds REST clients to talk to webhooks. It caches the clients
// to avoid duplicate creation.
// Originate from k8s.io/apiserver/pkg/admission/plugin/webhook/config.ClientManager
type ClientManager struct {
	authInfoResolver     webhook.AuthenticationInfoResolver
	serviceResolver      webhook.ServiceResolver
	negotiatedSerializer runtime.NegotiatedSerializer
	cache                *lru.Cache
}

// NewClientManager creates a ClientManager.
func NewClientManager() (ClientManager, error) {
	cache, err := lru.New(defaultCacheSize)
	if err != nil {
		return ClientManager{}, err
	}
	return ClientManager{
		cache: cache,
	}, nil
}

// SetAuthenticationInfoResolverWrapper sets the
// AuthenticationInfoResolverWrapper.
func (cm *ClientManager) SetAuthenticationInfoResolverWrapper(wrapper webhook.AuthenticationInfoResolverWrapper) {
	if wrapper != nil {
		cm.authInfoResolver = wrapper(cm.authInfoResolver)
	}
}

// SetAuthenticationInfoResolver sets the AuthenticationInfoResolver.
func (cm *ClientManager) SetAuthenticationInfoResolver(resolver webhook.AuthenticationInfoResolver) {
	cm.authInfoResolver = resolver
}

// SetServiceResolver sets the ServiceResolver.
func (cm *ClientManager) SetServiceResolver(sr webhook.ServiceResolver) {
	if sr != nil {
		cm.serviceResolver = sr
	}
}

// SetNegotiatedSerializer sets the NegotiatedSerializer.
func (cm *ClientManager) SetNegotiatedSerializer(n runtime.NegotiatedSerializer) {
	cm.negotiatedSerializer = n
}

// Validate checks if ClientManager is properly set up.
func (cm *ClientManager) Validate() error {
	var errs []error
	if cm.negotiatedSerializer == nil {
		errs = append(errs, fmt.Errorf("the ClientManager requires a negotiatedSerializer"))
	}
	if cm.serviceResolver == nil {
		errs = append(errs, fmt.Errorf("the ClientManager requires a serviceResolver"))
	}
	if cm.authInfoResolver == nil {
		errs = append(errs, fmt.Errorf("the ClientManager requires an authInfoResolver"))
	}
	return utilerrors.NewAggregate(errs)
}

// HookClient get a RESTClient from the cache, or constructs one based on the
// webhook configuration.
func (cm *ClientManager) HookClient(h *v1alpha1.Webhook) (*rest.RESTClient, error) {
	cacheKey, err := json.Marshal(h.ClientConfig)
	if err != nil {
		return nil, err
	}
	if client, ok := cm.cache.Get(string(cacheKey)); ok {
		return client.(*rest.RESTClient), nil
	}

	complete := func(cfg *rest.Config) (*rest.RESTClient, error) {
		cfg.UserAgent = controllerName // override default
		cfg.TLSClientConfig.CAData = h.ClientConfig.CABundle
		cfg.ContentConfig.NegotiatedSerializer = cm.negotiatedSerializer
		cfg.ContentConfig.ContentType = runtime.ContentTypeJSON
		client, err := rest.UnversionedRESTClientFor(cfg)
		if err == nil {
			cm.cache.Add(string(cacheKey), client)
		}
		return client, err
	}

	if svc := h.ClientConfig.Service; svc != nil {
		restConfig, err := cm.authInfoResolver.ClientConfigForService(svc.Name, svc.Namespace)
		if err != nil {
			return nil, err
		}
		cfg := rest.CopyConfig(restConfig)
		serverName := svc.Name + "." + svc.Namespace + ".svc"
		host := serverName + ":443"
		cfg.Host = "https://" + host
		if svc.Path != nil {
			cfg.APIPath = *svc.Path
		}
		cfg.TLSClientConfig.ServerName = serverName

		delegateDialer := cfg.Dial
		if delegateDialer == nil {
			delegateDialer = net.Dial
		}
		cfg.Dial = func(network, addr string) (net.Conn, error) {
			if addr == host {
				u, err := cm.serviceResolver.ResolveEndpoint(svc.Namespace, svc.Name)
				if err != nil {
					return nil, err
				}
				addr = u.Host
			}
			return delegateDialer(network, addr)
		}

		return complete(cfg)
	}

	if h.ClientConfig.URL == nil {
		return nil, &webhookerrors.ErrCallingWebhook{WebhookName: h.Name, Reason: ErrNeedServiceOrURL}
	}

	u, err := url.Parse(*h.ClientConfig.URL)
	if err != nil {
		return nil, &webhookerrors.ErrCallingWebhook{WebhookName: h.Name, Reason: fmt.Errorf("Unparsable URL: %v", err)}
	}

	restConfig, err := cm.authInfoResolver.ClientConfigFor(u.Host)
	if err != nil {
		return nil, err
	}

	cfg := rest.CopyConfig(restConfig)
	cfg.Host = u.Host
	cfg.APIPath = u.Path

	return complete(cfg)
}
