package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceFunction() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Raff serverless function's configuration (runtime, memory, " +
			"timeout, scaling). Code deploys happen outside Terraform — push with the " +
			"`raff deploy` CLI, the dashboard, or a connected GitHub repo. Settings " +
			"changes on a live function roll out as a new revision with zero downtime.",
		CreateContext: resourceFunctionCreate,
		ReadContext:   resourceFunctionRead,
		UpdateContext: resourceFunctionUpdate,
		DeleteContext: resourceFunctionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Display name, unique within the account.",
			},
			"runtime": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"python", "node", "go", "docker"}, false),
				Description:  "Managed runtime (`python`, `node`, `go`) or `docker` for a Dockerfile build.",
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "us-east",
				Description: "Deployment region.",
			},
			"memory_mb": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      256,
				ValidateFunc: validation.IntInSlice([]int{128, 256, 512, 1024, 2048, 4096}),
				Description:  "Instance memory. Memory is a paid meter (GB-seconds, wall-clock).",
			},
			"timeout_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     300,
				Description: "Request timeout. Up to 3600, or 86400 (24h) with extended_timeout.",
			},
			"extended_timeout": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Allow timeouts up to 24h (long runs bill wall-clock memory; no extra fee).",
			},
			"min_scale": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "0 = scale to zero (idle bills nothing); >=1 keeps warm instances at 30% of the memory rate.",
			},
			"max_scale": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     20,
				Description: "Maximum concurrent instances.",
			},
			// Computed
			"function_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Short reference id (fn-...).",
			},
			"slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL label — the name plus a short random suffix.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "HTTPS endpoint (automatic TLS).",
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"current_revision_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceFunctionCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	fn, _, err := client.Functions.Create(ctx, &raff.CreateFunctionRequest{
		Name:            d.Get("name").(string),
		Description:     d.Get("description").(string),
		Runtime:         d.Get("runtime").(string),
		Region:          d.Get("region").(string),
		MemoryMB:        d.Get("memory_mb").(int),
		TimeoutSeconds:  d.Get("timeout_seconds").(int),
		ExtendedTimeout: d.Get("extended_timeout").(bool),
		MinScale:        d.Get("min_scale").(int),
		MaxScale:        d.Get("max_scale").(int),
		SourceType:      "cli",
	})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fn.ID)
	return setFunctionState(d, fn)
}

func resourceFunctionRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	fn, _, err := client.Functions.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setFunctionState(d, fn)
}

func resourceFunctionUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.UpdateFunctionRequest{}
	changed := false
	if d.HasChange("description") {
		v := d.Get("description").(string)
		req.Description, changed = &v, true
	}
	if d.HasChange("memory_mb") {
		v := d.Get("memory_mb").(int)
		req.MemoryMB, changed = &v, true
	}
	if d.HasChange("timeout_seconds") {
		v := d.Get("timeout_seconds").(int)
		req.TimeoutSeconds, changed = &v, true
	}
	if d.HasChange("extended_timeout") {
		v := d.Get("extended_timeout").(bool)
		req.ExtendedTimeout, changed = &v, true
	}
	if d.HasChange("min_scale") {
		v := d.Get("min_scale").(int)
		req.MinScale, changed = &v, true
	}
	if d.HasChange("max_scale") {
		v := d.Get("max_scale").(int)
		req.MaxScale, changed = &v, true
	}
	if !changed {
		return resourceFunctionRead(ctx, d, meta)
	}

	fn, _, err := client.Functions.Update(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}
	return setFunctionState(d, fn)
}

func resourceFunctionDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.Functions.Delete(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setFunctionState(d *schema.ResourceData, fn *raff.Function) diag.Diagnostics {
	d.Set("name", fn.Name)
	d.Set("description", fn.Description)
	d.Set("runtime", fn.Runtime)
	d.Set("region", fn.Region)
	d.Set("memory_mb", fn.MemoryMB)
	d.Set("timeout_seconds", fn.TimeoutSeconds)
	d.Set("extended_timeout", fn.ExtendedTimeout)
	d.Set("min_scale", fn.MinScale)
	d.Set("max_scale", fn.MaxScale)
	d.Set("function_id", fn.FunctionID)
	d.Set("slug", fn.Slug)
	d.Set("url", fn.URL)
	d.Set("status", fn.Status)
	d.Set("current_revision_number", fn.CurrentRevisionNumber)
	d.Set("created_at", fn.CreatedAt)
	d.Set("updated_at", fn.UpdatedAt)
	return nil
}
