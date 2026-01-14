package main

import (
    "os"
    "github.com/r0niinx/huaweicloud"
)

func init() {
    if zoneType := os.Getenv("HUAWEI_ZONE_TYPE"); zoneType != "" {
        huaweicloud.DefaultZoneType = zoneType
    }

    if routerID := os.Getenv("HUAWEI_ROUTER_ID"); routerID != "" {
        huaweicloud.DefaultRouterID = routerID
    }

    if routerRegion := os.Getenv("HUAWEI_ROUTER_REGION"); routerRegion != "" {
        huaweicloud.DefaultRouterRegion = routerRegion
    }
}
