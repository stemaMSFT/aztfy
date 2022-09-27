package resourceset

import (
	"log"
	"sort"

	"github.com/magodo/armid"
	"github.com/magodo/aztft/aztft"
)

type AzureResourceSet struct {
	Resources []AzureResource
}

type AzureResource struct {
	Id         armid.ResourceId
	Properties map[string]interface{}

	// PesudoResourceInfo is only non-nil for the specially populated resources
	PesudoResourceInfo *PesudoResourceInfo
}

type PesudoResourceInfo struct {
	TFType string
	TFId   string
}

func (rset AzureResourceSet) ToTFResources() []TFResource {
	tfresources := []TFResource{}
	for _, res := range rset.Resources {
		// This is a TF pesudo resource, whose TF info are already available.
		if res.PesudoResourceInfo != nil {
			tfresources = append(tfresources, TFResource{
				AzureId: res.Id,
				TFId:    res.PesudoResourceInfo.TFId,
				TFType:  res.PesudoResourceInfo.TFType,
			})
			continue
		}

		azureId := res.Id.String()
		var (
			// Use the azure ID as the TF ID as a fallback
			tfId   = azureId
			tfType string
		)
		tftypes, tfids, err := aztft.QueryTypeAndId(azureId, true)
		if err == nil {
			if len(tfids) == 1 && len(tftypes) == 1 {
				tfId = tfids[0]
				tfType = tftypes[0]
			} else {
				log.Printf("WARNING: Expect one query result for resource type and TF id for %s, got %d type and %d id.\n", azureId, len(tftypes), len(tfids))
			}
		} else {
			log.Printf("WARNING: Failed to query resource type for %s: %v\n", azureId, err)
		}

		tfresources = append(tfresources, TFResource{
			AzureId: res.Id,
			TFId:    tfId,
			TFType:  tfType,
		})
	}

	sort.Slice(tfresources, func(i, j int) bool {
		return tfresources[i].AzureId.String() < tfresources[j].AzureId.String()
	})

	return tfresources
}