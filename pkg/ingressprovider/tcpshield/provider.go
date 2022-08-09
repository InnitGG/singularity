package tcpshield

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	"innit.gg/singularity/pkg/ingressprovider"
	"strconv"
)

const (
	Endpoint       = "https://api.tcpshield.com"
	ResourcePrefix = "singularity-"
)

var (
	ErrorDomainNotFound = errors.New("domain not found")
)

type provider struct {
	apiKey    string
	networkId uint32
	client    *fiber.Client
}

func (p *provider) Create(hostName string, backendSet []*ingressprovider.Backend) (string, error) {
	backendSetId, err := p.updateBackendSet(hostName, backendSet)
	if err != nil {
		return "", err
	}

	descriptor := &DomainDescriptor{
		Name:         hostName,
		BackendSetId: backendSetId,
		BAC:          false,
	}
	res := &DomainResponse{}
	code, _, errs := p.client.Post(fmt.Sprintf("%s/networks/%d/domains", Endpoint, p.networkId)).
		Add("X-API-Key", p.apiKey).
		JSON(descriptor).
		Struct(res)

	if len(errs) != 0 {
		return "", errs[0]
	}

	if code != 200 {
		return "", errors.Errorf("unexpected status code: %d", code)
	}

	if res.Data == nil {
		return "", errors.New("error creating domain")
	}

	return strconv.Itoa(int(res.Data.Id)), nil
}

func (p *provider) Update(hostName string, backendSet []*ingressprovider.Backend) error {
	var list DomainList
	code, _, errs := p.client.Get(fmt.Sprintf("%s/networks/%d/domains", Endpoint, p.networkId)).
		Add("X-API-Key", p.apiKey).
		Struct(&list)

	if len(errs) != 0 {
		return errs[0]
	}

	if code != 200 {
		return errors.Errorf("unexpected status code: %d", code)
	}

	var id uint32
	for _, domain := range list {
		if domain.Name == hostName {
			id = domain.Id
			break
		}
	}

	if id == 0 {
		return ErrorDomainNotFound
	}

	backendSetId, err := p.updateBackendSet(hostName, backendSet)
	if err != nil {
		return err
	}

	descriptor := &DomainDescriptor{
		Name:         hostName,
		BackendSetId: backendSetId,
		BAC:          false,
	}

	code, _, errs = p.client.Patch(fmt.Sprintf("%s/networks/%d/domains/%d", Endpoint, p.networkId, id)).
		Add("X-API-Key", p.apiKey).
		JSON(descriptor).
		Bytes()

	if len(errs) != 0 {
		return errs[0]
	}

	if code != 200 {
		return errors.Errorf("unexpected status code: %d", code)
	}

	return nil
}

func (p *provider) Delete(id string) error {
	// TODO: Delete backendset
	code, _, errs := p.client.Delete(fmt.Sprintf("%s/networks/%d/domains/%s", Endpoint, p.networkId, id)).
		Add("X-API-Key", p.apiKey).
		Bytes()

	if len(errs) != 0 {
		return errs[0]
	}

	if code != 200 {
		return errors.Errorf("unexpected status code: %d", code)
	}

	return nil
}

func CreateProvider(apiKey string, networkId uint32) ingressprovider.Provider {
	return &provider{
		apiKey:    apiKey,
		networkId: networkId,
		client:    fiber.AcquireClient(),
	}
}

func (p *provider) updateBackendSet(hostName string, backendSet []*ingressprovider.Backend) (uint32, error) {
	var list BackendSetList
	code, _, errs := p.client.Get(fmt.Sprintf("%s/networks/%d/backendSets", Endpoint, p.networkId)).
		Add("X-API-Key", p.apiKey).
		Struct(&list)

	if len(errs) != 0 {
		return 0, errs[0]
	}

	if code != 200 {
		return 0, errors.Errorf("unexpected status code: %d", code)
	}

	// Check if there is an existing backend set.
	var id uint32
	for _, set := range list {
		if set.Name == ResourcePrefix+hostName {
			id = set.Id
			break
		}
	}

	backends := convertBackendSet(backendSet)
	descriptor := &BackendSetDescriptor{
		Name:          ResourcePrefix + hostName,
		ProxyProtocol: false,
		Backends:      backends,
	}

	if id == 0 {
		// We need to create a new backend set.
		res := &BackendSetResponse{}
		code, _, errs = p.client.Post(fmt.Sprintf("%s/networks/%d/backendSets", Endpoint, p.networkId)).
			Add("X-API-Key", p.apiKey).
			JSON(descriptor).
			Struct(res)

		if code != 200 {
			return 0, errors.Errorf("unexpected status code: %d", code)
		}

		if res.Data == nil {
			return 0, errors.New("error creating backendset")
		}

		id = res.Data.Id
	} else {
		// We can update an existing backend set.
		code, _, errs = p.client.Patch(fmt.Sprintf("%s/networks/%d/backendSets/%d", Endpoint, p.networkId, id)).
			Add("X-API-Key", p.apiKey).
			JSON(descriptor).
			Bytes()

		if code != 200 {
			return 0, errors.Errorf("unexpected status code: %d", code)
		}
	}

	return id, nil
}

func convertBackendSet(set []*ingressprovider.Backend) []string {
	newSet := make([]string, len(set))
	for i, descriptor := range set {
		newSet[i] = fmt.Sprintf("%s:%d", descriptor.IP, descriptor.Port)
	}
	return newSet
}
