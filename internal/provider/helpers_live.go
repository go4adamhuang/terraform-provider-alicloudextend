package provider

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	liveclient "github.com/alibabacloud-go/live-20161101/v2/client"
)

// newLiveClientFromConfig creates a Live API client from ClientConfig.
func newLiveClientFromConfig(c *ClientConfig) (*liveclient.Client, error) {
	regionID := "cn-hangzhou"
	if c.Region != "" {
		regionID = c.Region
	}
	return liveclient.NewClient(&openapiutil.Config{
		AccessKeyId:     strPtr(c.AccessKeyID),
		AccessKeySecret: strPtr(c.AccessKeySecret),
		Endpoint:        strPtr("live.aliyuncs.com"),
		RegionId:        strPtr(regionID),
	})
}

// liveFunctionArg is used to serialise the Functions JSON for BatchSetLiveDomainConfigs.
type liveFunctionArg struct {
	ArgName  string `json:"argName"`
	ArgValue string `json:"argValue"`
}

type liveFunctionEntry struct {
	FunctionName string            `json:"functionName"`
	FunctionArgs []liveFunctionArg `json:"functionArgs"`
}

// batchSetConfig calls BatchSetLiveDomainConfigs for a single function.
// It retries up to 3 times on transient errors (e.g. InvalidStartEndTimeParameter)
// that can occur when the domain has just become online.
func batchSetConfig(live *liveclient.Client, domainName, functionName string, args map[string]string) error {
	fnArgs := make([]liveFunctionArg, 0, len(args))
	for k, v := range args {
		fnArgs = append(fnArgs, liveFunctionArg{ArgName: k, ArgValue: v})
	}
	entry := []liveFunctionEntry{{FunctionName: functionName, FunctionArgs: fnArgs}}
	b, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal functions JSON: %w", err)
	}

	const maxRetries = 3
	const retryInterval = 10 * time.Second
	for i := range maxRetries {
		_, err = live.BatchSetLiveDomainConfigs(&liveclient.BatchSetLiveDomainConfigsRequest{
			DomainNames: strPtr(domainName),
			Functions:   strPtr(string(b)),
		})
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "InvalidStartEndTimeParameter") && i < maxRetries-1 {
			time.Sleep(retryInterval)
			continue
		}
		return err
	}
	return err
}

// batchDeleteConfig calls BatchDeleteLiveDomainConfigs for one or more function names.
func batchDeleteConfig(live *liveclient.Client, domainName string, functionNames ...string) error {
	_, err := live.BatchDeleteLiveDomainConfigs(&liveclient.BatchDeleteLiveDomainConfigsRequest{
		DomainNames:   strPtr(domainName),
		FunctionNames: strPtr(strings.Join(functionNames, ",")),
	})
	return err
}

// describeFunctionArgs reads the args of a single function config back as a map.
// Returns nil map if no config exists.
func describeFunctionArgs(live *liveclient.Client, domainName, functionName string) (map[string]string, error) {
	resp, err := live.DescribeLiveDomainConfigs(&liveclient.DescribeLiveDomainConfigsRequest{
		DomainName:    strPtr(domainName),
		FunctionNames: strPtr(functionName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "InvalidDomain.NotFound") {
			return nil, nil
		}
		return nil, err
	}
	if resp.Body == nil || resp.Body.DomainConfigs == nil {
		return nil, nil
	}
	for _, cfg := range resp.Body.DomainConfigs.DomainConfig {
		if cfg.FunctionName != nil && *cfg.FunctionName == functionName && cfg.FunctionArgs != nil {
			m := make(map[string]string)
			for _, arg := range cfg.FunctionArgs.FunctionArg {
				if arg.ArgName != nil && arg.ArgValue != nil {
					m[*arg.ArgName] = *arg.ArgValue
				}
			}
			return m, nil
		}
	}
	return nil, nil
}

// describeAllFunctionConfigs returns all configs for a function (e.g. set_resp_header has many).
func describeAllFunctionConfigs(live *liveclient.Client, domainName, functionName string) ([]map[string]string, error) {
	resp, err := live.DescribeLiveDomainConfigs(&liveclient.DescribeLiveDomainConfigsRequest{
		DomainName:    strPtr(domainName),
		FunctionNames: strPtr(functionName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "InvalidDomain.NotFound") {
			return nil, nil
		}
		return nil, err
	}
	if resp.Body == nil || resp.Body.DomainConfigs == nil {
		return nil, nil
	}
	var result []map[string]string
	for _, cfg := range resp.Body.DomainConfigs.DomainConfig {
		if cfg.FunctionName != nil && *cfg.FunctionName == functionName && cfg.FunctionArgs != nil {
			m := make(map[string]string)
			for _, arg := range cfg.FunctionArgs.FunctionArg {
				if arg.ArgName != nil && arg.ArgValue != nil {
					m[*arg.ArgName] = *arg.ArgValue
				}
			}
			result = append(result, m)
		}
	}
	return result, nil
}
