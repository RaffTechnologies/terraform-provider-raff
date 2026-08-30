package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
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

		// Pool deletion drains and terminates every node in the pool, so the
		// delete call has to wait for that to finish before Terraform moves on.
		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

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
				Description: "Worker node plan ID from the `raff_k8s_node_plans` data source. Pools in one cluster may use different plans.",
			},
			"node_count": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Node count (1–20; the cluster keeps at least 2 workers overall). Scale-down drains nodes first, honouring PodDisruptionBudgets. With `autoscale_enabled = true` this is only the pool's starting size — the autoscaler owns the count from then on, and changing this value has no effect.",
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
		d.Set("status", string(p.Status))
		if p.PlanID != nil {
			d.Set("plan_id", *p.PlanID)
		}
		if p.AutoscaleEnabled != nil {
			d.Set("autoscale_enabled", *p.AutoscaleEnabled)
		}
		// On an autoscaling pool the autoscaler owns the node count, so the
		// live value is not drift and must not be written back to state:
		// refreshing it makes every later plan want to scale the pool back to
		// the configured number, which fights the autoscaler and churns nodes.
		if p.AutoscaleEnabled == nil || !*p.AutoscaleEnabled {
			d.Set("node_count", p.NodeCount)
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

	// node_count is the pool's size only while it is scaled manually. Once
	// autoscaling is on, the autoscaler decides the count from pending pods —
	// scaling to the configured number here would undo its decisions on every
	// apply (and a scale-down drains and destroys a node to do it).
	if d.HasChange("node_count") && !d.Get("autoscale_enabled").(bool) {
		if _, err := client.Kubernetes.ScaleNodePool(ctx, clusterID, d.Id(), d.Get("node_count").(int)); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceK8sNodePoolRead(ctx, d, meta)
}

func resourceK8sNodePoolDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	clusterID := d.Get("cluster_id").(string)
	if _, err := client.Kubernetes.DeleteNodePool(ctx, clusterID, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	// Deletion is asynchronous: the API accepts the request and the pool's
	// nodes are drained and terminated in the background. Return before that
	// finishes and Terraform moves straight on — on a plan_id change (ForceNew)
	// it creates the replacement immediately, and that create is rejected
	// because the old pool still holds the name. Wait for the pool to go.
	if err := waitForNodePoolGone(ctx, client, clusterID, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// waitForNodePoolGone polls until the pool is absent from the cluster's pool
// list (or reports status "deleted"). A pool that is already gone returns
// immediately.
func waitForNodePoolGone(ctx context.Context, client *raff.Client, clusterID, poolID string, timeout time.Duration) error {
	conf := &retry.StateChangeConf{
		Pending:    []string{"pending", "provisioning", "running", "scaling", "deleting"},
		Target:     []string{"deleted"},
		Timeout:    timeout,
		Delay:      5 * time.Second,
		MinTimeout: 5 * time.Second,
		Refresh: func() (any, string, error) {
			pools, _, err := client.Kubernetes.ListNodePools(ctx, clusterID)
			if err != nil {
				if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
					return poolID, "deleted", nil
				}
				return nil, "", err
			}
			for _, p := range pools {
				if p.ID.String() == poolID {
					return poolID, string(p.Status), nil
				}
			}
			return poolID, "deleted", nil
		},
	}
	if _, err := conf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for node pool %s to be deleted: %w", poolID, err)
	}
	return nil
}
