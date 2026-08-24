package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func dataSourceK8sVersions() *schema.Resource {
	return &schema.Resource{
		Description: "Available Kubernetes versions for cluster creation.",
		ReadContext: dataSourceK8sVersionsRead,
		Schema: map[string]*schema.Schema{
			"versions": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":         {Type: schema.TypeInt, Computed: true},
						"version":    {Type: schema.TypeString, Computed: true},
						"is_default": {Type: schema.TypeBool, Computed: true},
					},
				},
			},
			"default_version_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "ID of the platform default version.",
			},
		},
	}
}

func dataSourceK8sVersionsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	versions, _, err := client.Kubernetes.ListVersions(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	items := make([]any, 0, len(versions))
	defaultID := 0
	for _, v := range versions {
		item := map[string]any{}
		if v.ID != nil {
			item["id"] = *v.ID
		}
		if v.Version != nil {
			item["version"] = *v.Version
		}
		isDefault := v.IsDefault != nil && *v.IsDefault
		item["is_default"] = isDefault
		if isDefault && v.ID != nil {
			defaultID = *v.ID
		}
		items = append(items, item)
	}
	d.SetId("k8s-versions")
	d.Set("versions", items)
	d.Set("default_version_id", defaultID)
	return nil
}

func dataSourceK8sNodePlans() *schema.Resource {
	return &schema.Resource{
		Description: "Worker node plans with pricing for Kubernetes node pools.",
		ReadContext: dataSourceK8sNodePlansRead,
		Schema: map[string]*schema.Schema{
			"plans": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":              {Type: schema.TypeInt, Computed: true},
						"name":            {Type: schema.TypeString, Computed: true},
						"vcpu":            {Type: schema.TypeInt, Computed: true},
						"memory_gib":      {Type: schema.TypeInt, Computed: true},
						"ssd_gib":         {Type: schema.TypeInt, Computed: true},
						"price_per_month": {Type: schema.TypeFloat, Computed: true},
						"price_per_hour":  {Type: schema.TypeFloat, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceK8sNodePlansRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	plans, _, err := client.Kubernetes.ListNodePlans(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	items := make([]any, 0, len(plans.Plans))
	for _, p := range plans.Plans {
		items = append(items, map[string]any{
			"id":              p.ID,
			"name":            p.Name,
			"vcpu":            p.Vcpu,
			"memory_gib":      p.MemoryGib,
			"ssd_gib":         p.SsdGib,
			"price_per_month": float64(p.PricePerMonth),
			"price_per_hour":  float64(p.PricePerHour),
		})
	}
	d.SetId("k8s-node-plans")
	d.Set("plans", items)
	return nil
}

func dataSourceK8sCluster() *schema.Resource {
	return &schema.Resource{
		Description: "Reads an existing Kubernetes cluster by its short cluster ID.",
		ReadContext: dataSourceK8sClusterRead,
		Schema: map[string]*schema.Schema{
			"cluster_id":   {Type: schema.TypeString, Required: true, Description: "Short cluster ID."},
			"name":         {Type: schema.TypeString, Computed: true},
			"status":       {Type: schema.TypeString, Computed: true},
			"ready":        {Type: schema.TypeBool, Computed: true},
			"k8s_version":  {Type: schema.TypeString, Computed: true},
			"api_endpoint": {Type: schema.TypeString, Computed: true},
			"ha_enabled":   {Type: schema.TypeBool, Computed: true},
			"worker_count": {Type: schema.TypeInt, Computed: true},
			"kubeconfig": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func dataSourceK8sClusterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	clusterID := d.Get("cluster_id").(string)
	cluster, _, err := client.Kubernetes.Get(ctx, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(cluster.ClusterID)
	d.Set("name", cluster.Name)
	d.Set("status", string(cluster.Status))
	d.Set("ready", cluster.Ready)
	if cluster.K8SVersion != nil {
		d.Set("k8s_version", *cluster.K8SVersion)
	}
	if cluster.APIEndpoint != nil {
		d.Set("api_endpoint", *cluster.APIEndpoint)
	}
	if cluster.HaEnabled != nil {
		d.Set("ha_enabled", *cluster.HaEnabled)
	}
	if cluster.WorkerCount != nil {
		d.Set("worker_count", *cluster.WorkerCount)
	}
	if cluster.Status == raff.K8sClusterStatusRunning || cluster.Ready {
		if kc, _, err := client.Kubernetes.Kubeconfig(ctx, cluster.ClusterID); err == nil {
			d.Set("kubeconfig", kc.Kubeconfig)
		}
	}
	return nil
}
