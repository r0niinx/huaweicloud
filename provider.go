package huaweicloud

import (
	"context"
	"fmt"
	"sync"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Huawei Cloud
type Provider struct {
    AccessKeyId     string `json:"access_key_id,omitempty"`
    SecretAccessKey string `json:"secret_access_key,omitempty"`
    RegionId        string `json:"region_id,omitempty"`
    ProjectID       string `json:"project_id,omitempty"`
    CloudProvider   string `json:"cloud_provider,omitempty"`
    ZoneType        string `json:"zone_type,omitempty"`
    RouterID        string `json:"router_id,omitempty"`
    RouterRegion    string `json:"router_region,omitempty"`
    once            sync.Once
    client          *Client
}

func (p *Provider) init() {
    if p.ZoneType == "" {
        p.ZoneType = getEnvOrDefault("HUAWEI_ZONE_TYPE", "public")
    }
    if p.RouterID == "" {
        p.RouterID = os.Getenv("HUAWEI_ROUTER_ID")
    }
    if p.RouterRegion == "" {
        p.RouterRegion = getEnvOrDefault("HUAWEI_ROUTER_REGION", p.Region)
    }
}

func getEnvOrDefault(key, defaultValue string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultValue
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	client := p.getClient()

	records, err := client.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	var results []libdns.Record
	for _, record := range records {
		rec, err := record.libdnsRecord(zone)
		if err != nil {
			return nil, fmt.Errorf("parsing Huawei Cloud DNS record %+v: %v", record, err)
		}
		results = append(results, rec...)
	}

	return results, nil
}
// AppendRecords adds records to the zone. It returns the records that were added.
// NOTE: This implementation is NOT atomic.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := p.getClient()

	var results []libdns.Record
	for _, rec := range records {
		hwRec, err := hwRecord(zone, rec)
		if err != nil {
			return nil, fmt.Errorf("parsing libdns record %+v: %v", rec, err)
		}
		resp, err := client.AppendRecord(ctx, zone, hwRec)
		if err != nil {
			return nil, err
		}
		libdnsRecs, err := resp.libdnsRecord(zone)
		if err != nil {
			return nil, fmt.Errorf("parsing Huawei Cloud DNS record %+v: %v", resp, err)
		}
		results = append(results, libdnsRecs...)
	}

	return results, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
// NOTE: This implementation is NOT atomic.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := p.getClient()

	var results []libdns.Record
	for _, record := range records {
		rr := record.RR()
		id, err := client.GetRecordId(ctx, zone, rr.Name, rr.Type, rr.Data)
		if err != nil {
			// No existing record found, create a new one
			hwRec, err := hwRecord(zone, record)
			if err != nil {
				return nil, fmt.Errorf("parsing libdns record %+v: %v", record, err)
			}
			resp, err := client.AppendRecord(ctx, zone, hwRec)
			if err != nil {
				return nil, err
			}
			libdnsRecs, err := resp.libdnsRecord(zone)
			if err != nil {
				return nil, fmt.Errorf("parsing Huawei Cloud DNS record %+v: %v", resp, err)
			}
			results = append(results, libdnsRecs...)
		} else {
			// Existing record found, update it
			hwRec, err := hwRecord(zone, record)
			if err != nil {
				return nil, fmt.Errorf("parsing libdns record %+v: %v", record, err)
			}
			hwRec.Id = id
			hwRec.Ttl = int32(rr.TTL.Seconds())
			resp, err := client.UpdateRecord(ctx, zone, hwRec)
			if err != nil {
				return nil, err
			}
			libdnsRecs, err := resp.libdnsRecord(zone)
			if err != nil {
				return nil, fmt.Errorf("parsing Huawei Cloud DNS record %+v: %v", resp, err)
			}
			results = append(results, libdnsRecs...)
		}
	}

	return results, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
// NOTE: This implementation is NOT atomic.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := p.getClient()

	var results []libdns.Record
	for _, record := range records {
		rr := record.RR()
		id, err := client.GetRecordId(ctx, zone, rr.Name, rr.Type, rr.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to get record ID for %s: %v", rr.Name, err)
		}
		resp, err := client.DeleteRecord(ctx, zone, id)
		if err != nil {
			return nil, fmt.Errorf("failed to delete record %s: %v", rr.Name, err)
		}
		libdnsRecs, err := resp.libdnsRecord(zone)
		if err != nil {
			return nil, fmt.Errorf("parsing Huawei Cloud DNS record %+v: %v", resp, err)
		}
		results = append(results, libdnsRecs...)
	}

	return results, nil
}

// getClient initializes the client for the provider.
func (p *Provider) getClient() *Client {
    if p.client != nil {
        return p.client
    }

    if p.ZoneType == "" {
        p.ZoneType = getEnv("HUAWEI_ZONE_TYPE", "public")
    }
    if p.RouterID == "" {
        p.RouterID = os.Getenv("HUAWEI_ROUTER_ID")
    }
    if p.RouterRegion == "" {
        p.RouterRegion = getEnv("HUAWEI_ROUTER_REGION", p.Region)
    }
    p.client = &Client{
        AccessKeyId:     p.AccessKeyId,
        SecretAccessKey: p.SecretAccessKey,
        Region:          p.Region,
        ZoneType:        p.ZoneType,      // Передаем тип зоны
        RouterID:        p.RouterID,      // Передаем VPC ID
        RouterRegion:    p.RouterRegion,  // Передаем регион VPC
    }
    return p.client
}

func getEnv(key, defaultValue string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultValue
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
