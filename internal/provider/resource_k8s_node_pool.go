package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceK8sNodePool() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an additional node pool on a Raff Kubernetes cluster. The cluster's first pool is defined on `raff_k8s_cluster` (`default_pool`).",

		CreateContext: resourceK8sNodePoolCreate,
		ReadContext:   resourceK8sNodePoolRead,
		UpdateContext: resourceK8sNodePoolUpdate,
		DeleteContext: resourceK8sNodePoolDelete,

		Importer: &schema.ResourceImporter{
			// terraform import raff_k8s_node_pool.x <cluster-id>/<pool-id>
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
				parts := strings.SplitN(d.Id(), "/", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return nil, fmt.Errorf("import ID must be <cluster-id>/<pool-id>, got %q", d.Id())
				}
				d.Set("cluster_id", parts[0])
				d.SetId(parts[1])
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Short cluster ID (`raff_k8s_cluster.x.cluster_id`).",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Pool name, unique in the cluster.",
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
				Description: "Node count (2–20). Scale-down drains nodes first, honouring PodDisruptionBudgets.",
			},
			"autoscale_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable autoscaling between `min_nodes` and `max_nodes`.",
			},
			"min_nodes": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Autoscale lower bound.",
			},
			"max_nodes": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Autoscale upper bound.",
			},
			"labels": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "JSON object of Kubernetes labels applied to every node in the pool, e.g. `{\"tier\":\"db\"}`.",
			},
			"taints": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "JSON array of taints, e.g. `[{\"key\":\"gpu\",\"value\":\"true\",\"effect\":\"NoSchedule\"}]`.",
			},
			// Computed
			"status":            {Type: schema.TypeString, Computed: true},
			"actual_node_count": {Type: schema.TypeInt, Computed: true, Description: "Nodes that currently exist (differs from node_count briefly while scaling)."},
		},
	}
}

func resourceK8sNodePoolCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	planID := d.Get("plan_id").(int)
	req := &raff.AddK8sNodePoolRequest{
		Name:      d.Get("name").(string),
		NodeCount: d.Get("node_count").(int),
		PlanID:    &planID,
	}
	applyPoolOptions(d, &req.AutoscaleEnabled, &req.MinNodes, &req.MaxNodes, &req.Labels, &req.Taints)

	pool, _, err := client.Kubernetes.AddNodePool(ctx, d.Get("cluster_id").(string), req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(pool.ID.String())
	return resourceK8sNodePoolRead(ctx, d, meta)
}

func applyPoolOptions(d *schema.ResourceData, autoscale **bool, minNodes, maxNodes **int, labels, taints **string) {
	if d.Get("autoscale_enabled").(bool) {
		t := true
		*autoscale = &t
	}
	if v, ok := d.GetOk("min_nodes"); ok {
		n := v.(int)
		*minNodes = &n
	}
	if v, ok := d.GetOk("max_nodes"); ok {
		n := v.(int)
		*maxNodes = &n
	}
	if v, ok := d.GetOk("labels"); ok {
		s := v.(string)
		*labels = &s
	}
	if v, ok := d.GetOk("taints"); ok {
		s := v.(string)
		*taints = &s
	}
}

func resourceK8sNodePoolRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	pools, _, err := client.Kubernetes.ListNodePools(ctx, d.Get("cluster_id").(string))
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	for _, p := range pools {
		if p.ID.String() != d.Id() {
			continue
		}
		d.Set("name", p.Name)
		d.Set("node_count", p.NodeCount)
		d.Set("status", string(p.Status))
		if p.PlanID != nil {
			d.Set("plan_id", *p.PlanID)
		}
		if p.AutoscaleEnabled != nil {
			d.Set("autoscale_enabled", *p.AutoscaleEnabled)
		}
		if p.MinNodes != nil {
			d.Set("min_nodes", *p.MinNodes)
		}
		if p.MaxNodes != nil {
			d.Set("max_nodes", *p.MaxNodes)
		}
		if p.Labels != nil {
			d.Set("labels", *p.Labels)
		}
		if p.Taints != nil {
			d.Set("taints", *p.Taints)
		}
		if p.ActualNodeCount != nil {
			d.Set("actual_node_count", *p.ActualNodeCount)
		}
		return nil
	}
	d.SetId("")
	return nil
}

func resourceK8sNodePoolUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	clusterID := d.Get("cluster_id").(string)

	if d.HasChanges("autoscale_enabled", "min_nodes", "max_nodes", "labels", "taints") {
		req := &raff.UpdateK8sNodePoolRequest{}
		autoscale := d.Get("autoscale_enabled").(bool)
		req.AutoscaleEnabled = &autoscale
		if v, ok := d.GetOk("min_nodes"); ok {
			n := v.(int)
			req.MinNodes = &n
		}
		if v, ok := d.GetOk("max_nodes"); ok {
			n := v.(int)
			req.MaxNodes = &n
		}
		if v, ok := d.GetOk("labels"); ok {
			s := v.(string)
			req.Labels = &s
		}
		if v, ok := d.GetOk("taints"); ok {
			s := v.(string)
			req.Taints = &s
		}
		if _, _, err := client.Kubernetes.UpdateNodePool(ctx, clusterID, d.Id(), req); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("node_count") {
		if _, err := client.Kubernetes.ScaleNodePool(ctx, clusterID, d.Id(), d.Get("node_count").(int)); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceK8sNodePoolRead(ctx, d, meta)
}

func resourceK8sNodePoolDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.Kubernetes.DeleteNodePool(ctx, d.Get("cluster_id").(string), d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
