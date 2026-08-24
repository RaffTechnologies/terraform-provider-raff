package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceK8sCluster() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Raff managed Kubernetes cluster. The cluster is created with one default node pool; add more pools with `raff_k8s_node_pool`. Only pay-as-you-go accounts can create clusters through the API.",

		CreateContext: resourceK8sClusterCreate,
		ReadContext:   resourceK8sClusterRead,
		UpdateContext: resourceK8sClusterUpdate,
		DeleteContext: resourceK8sClusterDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cluster name (1–63 lowercase letters, digits or hyphens, unique per account). Renames in place.",
			},
			"version_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Kubernetes version ID from the `raff_k8s_versions` data source. Defaults to the platform default.",
			},
			"ha_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "HA control plane (3 masters + redundant gateway, flat monthly fee). Can be upgraded in place from `false` to `true`; downgrading is not supported.",
			},
			"default_pool": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "The cluster's default node pool, created with the cluster.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "default-pool",
							ForceNew:    true,
							Description: "Pool name.",
						},
						"plan_id": {
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    true,
							Description: "Worker node plan ID from the `raff_k8s_node_plans` data source.",
						},
						"node_count": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Worker node count (2–20). Scale-down drains nodes first.",
						},
						"id": {Type: schema.TypeString, Computed: true, Description: "Pool ID."},
					},
				},
			},
			"traefik_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				Description: "Install the Traefik ingress controller.",
			},
			"metallb_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				Description: "Install MetalLB for `LoadBalancer` services.",
			},
			"storage_node_count": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				ForceNew:    true,
				Description: "Dedicated Longhorn storage nodes (0, 2 or 3). 0 disables in-cluster block storage.",
			},
			"storage_node_disk_gb": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Data disk per storage node in GB. Required when `storage_node_count` > 0.",
			},
			"cluster_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Pod network CIDR. Defaults to 10.42.0.0/16.",
			},
			"service_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Service network CIDR. Defaults to 10.43.0.0/16.",
			},
			// Computed
			"cluster_id":   {Type: schema.TypeString, Computed: true, Description: "Short cluster ID used in URLs and the API endpoint hostname."},
			"status":       {Type: schema.TypeString, Computed: true},
			"ready":        {Type: schema.TypeBool, Computed: true},
			"k8s_version":  {Type: schema.TypeString, Computed: true},
			"api_endpoint": {Type: schema.TypeString, Computed: true},
			"vpc_id":       {Type: schema.TypeString, Computed: true},
			"kubeconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Admin kubeconfig YAML for kubectl/Helm providers.",
			},
			"price_per_month": {Type: schema.TypeFloat, Computed: true},
		},
	}
}

func resourceK8sClusterCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	pool := d.Get("default_pool").([]any)[0].(map[string]any)
	req := &raff.CreateK8sClusterRequest{
		Name: d.Get("name").(string),
		NodePools: []raff.K8sNodePoolInput{{
			Name:      pool["name"].(string),
			PlanID:    pool["plan_id"].(int),
			NodeCount: pool["node_count"].(int),
		}},
	}
	if v, ok := d.GetOk("version_id"); ok {
		id := v.(int)
		req.K8SVersionID = &id
	}
	if d.Get("ha_enabled").(bool) {
		ha := true
		req.HaEnabled = &ha
	}
	traefik := d.Get("traefik_enabled").(bool)
	req.TraefikEnabled = &traefik
	metallb := d.Get("metallb_enabled").(bool)
	req.MetallbEnabled = &metallb
	if v, ok := d.GetOk("storage_node_count"); ok && v.(int) > 0 {
		count := raff.CreateK8sClusterStorageNodeCount(v.(int))
		req.StorageNodeCount = &count
		if disk, ok := d.GetOk("storage_node_disk_gb"); ok {
			g := disk.(int)
			req.StorageNodeDiskGb = &g
		}
	}
	if v, ok := d.GetOk("cluster_cidr"); ok {
		c := v.(string)
		req.ClusterCidr = &c
	}
	if v, ok := d.GetOk("service_cidr"); ok {
		c := v.(string)
		req.ServiceCidr = &c
	}

	cluster, _, err := client.Kubernetes.Create(ctx, req, "")
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(cluster.ClusterID)

	waitCtx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()
	if _, err := client.Kubernetes.WaitForStatus(waitCtx, cluster.ClusterID, raff.K8sClusterStatusRunning); err != nil {
		return diag.Errorf("cluster %s did not reach running: %s", cluster.ClusterID, err)
	}
	return resourceK8sClusterRead(ctx, d, meta)
}

func resourceK8sClusterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	cluster, _, err := client.Kubernetes.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("name", cluster.Name)
	d.Set("cluster_id", cluster.ClusterID)
	d.Set("status", string(cluster.Status))
	d.Set("ready", cluster.Ready)
	if cluster.K8SVersionID != nil {
		d.Set("version_id", *cluster.K8SVersionID)
	}
	if cluster.K8SVersion != nil {
		d.Set("k8s_version", *cluster.K8SVersion)
	}
	if cluster.APIEndpoint != nil {
		d.Set("api_endpoint", *cluster.APIEndpoint)
	}
	if cluster.HaEnabled != nil {
		d.Set("ha_enabled", *cluster.HaEnabled)
	}
	if cluster.TraefikEnabled != nil {
		d.Set("traefik_enabled", *cluster.TraefikEnabled)
	}
	if cluster.MetallbEnabled != nil {
		d.Set("metallb_enabled", *cluster.MetallbEnabled)
	}
	if cluster.VpcID != nil {
		d.Set("vpc_id", *cluster.VpcID)
	}
	if cluster.PricePerMonth != nil {
		d.Set("price_per_month", *cluster.PricePerMonth)
	}

	// The default pool is the first pool created with the cluster; match by
	// the stored pool ID when known, else the pool named like the config.
	if cluster.NodePools != nil && len(*cluster.NodePools) > 0 {
		var stored string
		if v := d.Get("default_pool").([]any); len(v) > 0 {
			stored, _ = v[0].(map[string]any)["id"].(string)
		}
		for _, p := range *cluster.NodePools {
			if (stored != "" && p.ID.String() == stored) || (stored == "" && p.Name == defaultPoolName(d)) {
				pool := map[string]any{
					"id":         p.ID.String(),
					"name":       p.Name,
					"node_count": p.NodeCount,
				}
				if p.PlanID != nil {
					pool["plan_id"] = *p.PlanID
				}
				d.Set("default_pool", []any{pool})
				break
			}
		}
	}

	// Kubeconfig is only available once the cluster runs.
	if cluster.Status == raff.K8sClusterStatusRunning || cluster.Ready {
		if kc, _, err := client.Kubernetes.Kubeconfig(ctx, d.Id()); err == nil {
			d.Set("kubeconfig", kc.Kubeconfig)
		}
	}
	return nil
}

func defaultPoolName(d *schema.ResourceData) string {
	if v := d.Get("default_pool").([]any); len(v) > 0 {
		if name, ok := v[0].(map[string]any)["name"].(string); ok && name != "" {
			return name
		}
	}
	return "default-pool"
}

func resourceK8sClusterUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	if d.HasChange("name") {
		if _, err := client.Kubernetes.Rename(ctx, d.Id(), d.Get("name").(string)); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("ha_enabled") {
		old, current := d.GetChange("ha_enabled")
		if old.(bool) && !current.(bool) {
			return diag.Errorf("ha_enabled cannot be disabled once enabled")
		}
		if current.(bool) {
			if _, err := client.Kubernetes.UpgradeHA(ctx, d.Id()); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("default_pool.0.node_count") {
		poolID, _ := d.Get("default_pool.0.id").(string)
		if poolID == "" {
			return diag.Errorf("default pool ID unknown — run 'terraform refresh' first")
		}
		count := d.Get("default_pool.0.node_count").(int)
		if _, err := client.Kubernetes.ScaleNodePool(ctx, d.Id(), poolID, count); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceK8sClusterRead(ctx, d, meta)
}

func resourceK8sClusterDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.Kubernetes.Delete(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	defer cancel()
	if err := client.Kubernetes.WaitForDeleted(waitCtx, d.Id()); err != nil {
		return diag.Errorf("cluster %s deletion did not complete: %s", d.Id(), err)
	}
	return nil
}
